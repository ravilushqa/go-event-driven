package main

import (
	"context"
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
	"tickets/handler/pubsub"
	"tickets/pkg"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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

	watermillRouter := pubsub.NewWatermillRouter(receiptsClient, spreadsheetsClient, redisClient, watermillLogger)

	httpServer := httpHandler.NewServer(redisPublisher, spreadsheetsClient, ":8080")

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return watermillRouter.Run(ctx)
	})

	g.Go(func() error {
		// we don't want to start HTTP server before Watermill router (so service won't be healthy before it's ready)
		<-watermillRouter.Running()

		err := httpServer.Run(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	err = g.Wait()
	if err != nil {
		log.FromContext(ctx).WithError(err).Error("error while running the service")
	}
}
