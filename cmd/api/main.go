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
	"github.com/redis/go-redis/v9"
	"github.com/saarwasserman/auth/internal/data"
	"github.com/saarwasserman/auth/internal/jsonlog"
	"github.com/saarwasserman/auth/internal/vcs"
	"google.golang.org/grpc"

	middlewareAuth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"github.com/saarwasserman/auth/protogen/auth"
	"github.com/saarwasserman/auth/protogen/notifications"
)

var (
	version = vcs.Version()
)

type config struct {
	port    int
	env     string
	session struct {
		inactivityTime int
	}
	db struct {
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
		enabled bool
		host    string
		port    int
	}
	cors struct {
		trustedOrigins []string
	}
	cache struct {
		enabled  bool
		endpoint string
	}
}

type application struct {
	auth.UnimplementedAuthenticationServer
	config   config
	logger   *jsonlog.Logger
	models   data.Models
	notifier notifications.NotificationsClient
	cache    *redis.Client
}

func main() {
	var cfg config

	// server
	flag.IntVar(&cfg.port, "port", 40020, "API Server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// session
	flag.IntVar(&cfg.session.inactivityTime, "session-inactivity-time", 5, "User inactivity duration in minutes")

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
	flag.BoolVar(&cfg.notificationsService.enabled, "notifications-enabled", false, "enable notifications through notification service")

	// cache
	flag.StringVar(&cfg.cache.endpoint, "cache-endpoint", os.Getenv("CACHE_ENDPOINT"), "Cache Endpoint")
	flag.BoolVar(&cfg.cache.enabled, "cache-enabled", false, "enable caching")

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

	// setup notifications service client
	var notifier notifications.NotificationsClient
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if !cfg.notificationsService.enabled {
		logger.PrintInfo("notifications service is disabled", nil)
	} else {
		logger.PrintInfo("connecting to notifications service...", nil)
		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", cfg.notificationsService.host, cfg.notificationsService.port), opts...)
		if err != nil {
			logger.PrintFatal(err, nil)
			return
		}

		defer conn.Close()

		notifier = notifications.NewNotificationsClient(conn)
	}

	// setup cache
	var cache *redis.Client
	if !cfg.cache.enabled {
		logger.PrintInfo("caching is disabled", nil)
	} else {
		logger.PrintInfo("enabling caching...", nil)
		cache = redis.NewClient(&redis.Options{
			Addr: cfg.cache.endpoint,
		})

		defer cache.Close()

		// test cache connection
		_, err := cache.Ping(context.Background()).Result()
		if err != nil {
			logger.PrintFatal(err, nil)
			return
		}
	}

	app := &application{
		config:   cfg,
		logger:   logger,
		models:   data.NewModels(db),
		notifier: notifier,
		cache:    cache,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", app.config.port))
	if err != nil {
		app.logger.PrintFatal(err, nil)
		return
	}

	serviceRegistrar := grpc.NewServer(grpc.ChainUnaryInterceptor(
		// authentication
		selector.UnaryServerInterceptor(
			middlewareAuth.UnaryServerInterceptor(app.Authenticator),
			selector.MatchFunc(app.AuthMatcher),
		),
	))

	app.logger.PrintInfo(fmt.Sprintf("listening on %s", listener.Addr().String()), nil)
	auth.RegisterAuthenticationServer(serviceRegistrar, app)
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
