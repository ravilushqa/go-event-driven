package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"tickets/gateway"
	httpHandler "tickets/handler/http"
	"tickets/handler/subscriber"
	"tickets/pkg"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Init(logrus.InfoLevel)

	c, err := clients.NewClients(os.Getenv("GATEWAY_ADDR"), func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
		return nil
	})
	if err != nil {
		panic(err)
	}

	receiptsClient := gateway.NewReceiptsClient(c)
	spreadsheetsClient := gateway.NewSpreadsheetsClient(c)
	watermillLogger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))
	redisClient := pkg.NewRedisClient(os.Getenv("REDIS_ADDR"))

	redisPublisher := pkg.NewRedisPublisher(redisClient, watermillLogger)

	watermillRouter := subscriber.NewWatermillRouter(
		receiptsClient,
		spreadsheetsClient,
		redisClient,
		watermillLogger,
	)

	echoRouter := httpHandler.NewHttpRouter(
		redisPublisher,
		spreadsheetsClient,
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return watermillRouter.Run(ctx)
	})

	g.Go(func() error {
		// we don't want to start HTTP server before Watermill router (so service won't be healthy before it's ready)
		<-watermillRouter.Running()

		err := echoRouter.Start(":8080")
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		return echoRouter.Shutdown(context.Background())
	})

	err = g.Wait()
	if err != nil {
		log.FromContext(ctx).WithError(err).Error("error while running the service")
	}
}
