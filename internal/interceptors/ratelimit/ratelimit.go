package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/realip"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/// RATELIMIT ///

type RateLimiter struct {
	rps     float64
	burst   int
	enabled bool
}

func NewRateLimiter(rps float64, burst int, enabled bool) *RateLimiter {
	return &RateLimiter{
		rps: rps,
		burst: burst,
		enabled: enabled,
	}
}

func (limiter *RateLimiter) Limit(ctx context.Context) error {

	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	if limiter.enabled {
		go func() {
			for {
				time.Sleep(time.Minute)
				mu.Lock()

				for ip, client := range clients {
					if time.Since(client.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}

				mu.Unlock()
			}
		}()

		addr, _ := realip.FromContext(ctx)
		ip := addr.String()

		mu.Lock()

		if _, found := clients[ip]; !found {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(limiter.rps), limiter.burst)}
		}

		clients[ip].lastSeen = time.Now()

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			return status.Error(codes.ResourceExhausted, "client reached request limit")
		}

		mu.Unlock()
	}

	return nil
}


func UnaryServerInterceptor(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := limiter.Limit(ctx); err != nil {
   			return nil, err
  		}
  	
		return handler(ctx, req)
	}
}
