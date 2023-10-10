package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jessevdk/go-flags"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"tickets/db"
	"tickets/gateway"
	httpHandler "tickets/handler/http"
	"tickets/handler/pubsub"
	"tickets/pkg"
)

var opts struct {
	Mock        bool   `long:"mock" env:"MOCK" description:"Mock external services"`
	HTTPAddress string `long:"http-addr" env:"HTTP_ADDR" default:":8080" description:"HTTP address to listen on"`
	GatewayAddr string `long:"gateway-addr" env:"GATEWAY_ADDR" description:"Gateway address"`
	RedisAddr   string `long:"redis-addr" env:"REDIS_ADDR" default:"localhost:8080" description:"Redis address"`
	PostgresURL string `long:"postgres-url" env:"POSTGRES_URL" default:"postgres://user:password@localhost:5432/db?sslmode=disable" description:"Postgres URL"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Init(logrus.InfoLevel)
	logger := logrus.NewEntry(logrus.StandardLogger())

	_, err := flags.Parse(&opts)
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			panic(err)
		}
		return
	}

	c, err := clients.NewClients(opts.GatewayAddr, func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))
		return nil
	})
	if err != nil {
		panic(err)
	}

	dbconn, err := sqlx.Open("postgres", opts.PostgresURL)
	if err != nil {
		panic(err)
	}
	defer dbconn.Close()

	err = db.InitializeDatabaseSchema(dbconn)
	if err != nil {
		panic(err)
	}

	receiptsClient := gateway.NewReceiptsClient(c)
	spreadsheetsClient := gateway.NewSpreadsheetsClient(c)
	watermillLogger := log.NewWatermill(logger)
	redisClient := pkg.NewRedisClient(opts.RedisAddr)
	defer redisClient.Close()

	publisher := pkg.NewRedisPublisher(redisClient, watermillLogger)

	watermillRouter, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		panic(err)
	}
	pubsub.UseMiddlewares(watermillRouter, watermillLogger)

	eventBus, err := pkg.NewEventBus(publisher)
	if err != nil {
		panic(err)
	}

	err = pkg.RegisterEventHandlers(
		redisClient,
		watermillRouter,
		[]cqrs.EventHandler{
			pubsub.IssueReceiptHandler(receiptsClient),
			pubsub.AppendToTrackerHandler(spreadsheetsClient),
			pubsub.CancelTicketHandler(spreadsheetsClient),
		},
		watermillLogger,
	)
	if err != nil {
		panic(err)
	}

	httpServer := httpHandler.NewServer(eventBus, spreadsheetsClient, opts.HTTPAddress)

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
