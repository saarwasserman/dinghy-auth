package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"saarwasserman.com/auth/internal/data"
	"saarwasserman.com/auth/internal/jsonlog"
	"saarwasserman.com/auth/internal/vcs"

	notifications "saarwasserman.com/auth/grpcgen/notifications/proto"
	users "saarwasserman.com/auth/grpcgen/users/proto"
)

var (
	version = vcs.Version()
)

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	notificationsService struct {
		host     string
		port     int
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	users.UnimplementedUsersServiceServer
	config config
	logger *jsonlog.Logger
	models data.Models
	notifier notifications.EMailServiceClient
}

func main() {
	var cfg config

	// server
	flag.IntVar(&cfg.port, "port", 40020, "API Server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// db
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("AUTH_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	// limiter
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	// notifications service
	flag.StringVar(&cfg.notificationsService.host, "notifications-service-host", "localhost", "notifications service host")
	flag.IntVar(&cfg.notificationsService.port, "notifications-service-port", 40010, "notifications service port")

	// cors
	flag.Func("cors-trusted-origins", "Trusted CORS Origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	} else {
		logger.PrintInfo("database connection pool established", nil)
	}

	defer db.Close()

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutins", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.notificationsService.host, cfg.notificationsService.port), opts...)
	if err != nil {
		logger.PrintFatal(err, nil)
		return
	}

	defer conn.Close()

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		notifier: notifications.NewEMailServiceClient(conn),
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", app.config.port))
	if err != nil {
		app.logger.PrintFatal(err, nil)
		return
	}

	serviceRegistrar := grpc.NewServer()

	app.logger.PrintInfo(fmt.Sprintf("listening on %s", listener.Addr().String()), nil)
	users.RegisterUsersServiceServer(serviceRegistrar, app)
	// reflection.Register(serviceRegistrar)
	err = serviceRegistrar.Serve(listener)
	if err != nil {
		log.Fatalf("cannot serve %s", err)
		return
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
