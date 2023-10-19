package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/jessevdk/go-flags"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"tickets/gateway"
	"tickets/service"
)

var opts struct {
	Mock        bool   `long:"mock" env:"MOCK" description:"Mock external services"`
	HTTPAddress string `long:"http-addr" env:"HTTP_ADDR" default:":8080" description:"HTTP address to listen on"`
	GatewayAddr string `long:"gateway-addr" env:"GATEWAY_ADDR" default:"http://localhost:8888" description:"Gateway address"`
	RedisAddr   string `long:"redis-addr" env:"REDIS_ADDR" default:"localhost:6379" description:"Redis address"`
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

	receiptsClient := gateway.NewReceiptsClient(c)
	spreadsheetsClient := gateway.NewSpreadsheetsClient(c)
	filesClient := gateway.NewFilesClient(c)
	deadNationClient := gateway.NewDeadNationClient(c)
	paymentClient := gateway.NewPaymentClient(c)
	redisClient := redis.NewClient(&redis.Options{Addr: opts.RedisAddr})
	defer redisClient.Close()

	err = service.New(opts.HTTPAddress, dbconn, redisClient, spreadsheetsClient, receiptsClient, filesClient, deadNationClient, paymentClient).Run(ctx)
	if err != nil {
		logger.WithError(err).Error("service failed")
	}
}
