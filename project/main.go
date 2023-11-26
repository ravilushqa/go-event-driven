package main

import (
	"context"
	"fmt"
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
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"tickets/app"
	"tickets/gateway"
	"tickets/tracing"
)

var opts struct {
	Mock           bool   `long:"mock" env:"MOCK" description:"Mock external services"`
	HTTPAddress    string `long:"http-addr" env:"HTTP_ADDR" default:":8080" description:"HTTP address to listen on"`
	GatewayAddr    string `long:"gateway-addr" env:"GATEWAY_ADDR" default:"http://localhost:8888" description:"Gateway address"`
	RedisAddr      string `long:"redis-addr" env:"REDIS_ADDR" default:"localhost:6379" description:"Redis address"`
	PostgresURL    string `long:"postgres-url" env:"POSTGRES_URL" default:"postgres://user:password@localhost:5432/db?sslmode=disable" description:"Postgres URL"`
	JaegerEndpoint string `long:"jaeger-endpoint" env:"JAEGER_ENDPOINT" default:"http://localhost:14268/api/traces" description:"Jaeger endpoint"`
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

	traceProvider := tracing.ConfigureTraceProvider(opts.JaegerEndpoint, opts.GatewayAddr)
	defer traceProvider.Shutdown(ctx)

	traceHttpClient := &http.Client{Transport: otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("HTTP %s %s %s", r.Method, r.URL.String(), operation)
		}),
	)}

	apiClients, err := clients.NewClientsWithHttpClient(
		opts.GatewayAddr,
		func(ctx context.Context, req *http.Request) error {
			return nil
		},
		traceHttpClient,
	)
	if err != nil {
		panic(err)
	}

	traceDB, err := otelsql.Open("postgres", opts.PostgresURL,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithDBName("db"))
	if err != nil {
		panic(err)
	}

	db := sqlx.NewDb(traceDB, "postgres")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	receiptsClient := gateway.NewReceiptsClient(apiClients)
	spreadsheetsClient := gateway.NewSpreadsheetsClient(apiClients)
	filesClient := gateway.NewFilesClient(apiClients)
	deadNationClient := gateway.NewDeadNationClient(apiClients)
	paymentClient := gateway.NewPaymentClient(apiClients)
	transClient := gateway.NewTransportationClient(apiClients)
	redisClient := redis.NewClient(&redis.Options{Addr: opts.RedisAddr})
	defer redisClient.Close()

	err = app.New(
		opts.HTTPAddress,
		db,
		redisClient,
		spreadsheetsClient,
		receiptsClient,
		filesClient,
		deadNationClient,
		paymentClient,
		transClient,
		traceProvider,
	).Run(ctx)
	if err != nil {
		logger.WithError(err).Error("app failed")
	}
}
