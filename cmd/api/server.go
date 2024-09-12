package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	middlewareAuth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/realip"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"github.com/saarwasserman/auth/internal/interceptors/ratelimit"
	"github.com/saarwasserman/auth/protogen/auth"
	"google.golang.org/grpc"
)

func (app *application) serve() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", app.config.port))
	if err != nil {
		app.logger.PrintFatal(err, nil)
		return err
	}

	// realip
	headers := []string{realip.XForwardedFor, realip.XRealIp}
	realIpOpts := []realip.Option{
		realip.WithHeaders(headers),
		realip.WithTrustedProxiesCount(1),
	}

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(
		// authentication
		selector.UnaryServerInterceptor(
			middlewareAuth.UnaryServerInterceptor(app.Authenticator),
			selector.MatchFunc(app.AuthMatcher),
		),
		// ratelimit
		selector.UnaryServerInterceptor(
			ratelimit.UnaryServerInterceptor(app.limiter),
			selector.MatchFunc(app.RatelimitMatcher),
		),
		// realip
		realip.UnaryServerInterceptorOpts(realIpOpts...),
	))

	app.logger.PrintInfo(fmt.Sprintf("listening on %s", listener.Addr().String()), nil)
	auth.RegisterAuthenticationServer(srv, app)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.PrintInfo("caught signal", map[string]string{
			"signal": s.String(),
		})


		app.logger.PrintInfo("completing background tasks", map[string]string{"addr": listener.Addr().String()})

		app.wg.Wait()

		srv.GracefulStop()
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": listener.Addr().String(),
		"env":  app.config.env,
	})

	err = srv.Serve(listener)
	print(err)
	if err != nil {
		return err
	}

	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": listener.Addr().String(),
	})

	return nil
}
