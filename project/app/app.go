package app

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"

	dbLib "tickets/db"
	"tickets/entity"
	"tickets/http"
	migrations "tickets/migration"
	"tickets/pubsub"
	"tickets/pubsub/bus"
	"tickets/pubsub/command"
	"tickets/pubsub/event"
	"tickets/pubsub/outbox"
	"tickets/tracing"
)

var (
	veryImportantCounter = promauto.NewCounter(prometheus.CounterOpts{
		// metric will be named tickets_very_important_counter_total
		Namespace: "tickets",
		Name:      "very_important_counter_total",
		Help:      "Total number of very important things processed",
	})
)

func init() {
	log.Init(logrus.InfoLevel)
}

type App struct {
	db              *sqlx.DB
	watermillRouter *message.Router
	httpServer      *http.Server
	bookingHandlers event.OpsBookingHandlers
	dataLake        dbLib.DataLake
	traceProvider   *tracesdk.TracerProvider
}

func New(
	addr string,
	db *sqlx.DB,
	redisClient *redis.Client,
	spreadsheetsService event.SpreadsheetsAPI,
	receiptsService event.ReceiptsService,
	fileService event.FileService,
	deadNationService event.DeadNationService,
	paymentService event.PaymentService,
	transService command.TransportationService,
	traceProvider *tracesdk.TracerProvider,
) App {
	var redisPublisher message.Publisher

	watermillLogger := log.NewWatermill(log.FromContext(context.Background()))
	redisPublisher = pubsub.NewRedisPublisher(redisClient, watermillLogger)
	redisPublisher = tracing.PublisherDecorator{Publisher: redisPublisher}

	eventBus, err := bus.NewEventBus(redisPublisher)
	if err != nil {
		panic(fmt.Errorf("failed to create event bus: %w", err))
	}

	ticketsRepo := dbLib.NewTicketsPostgresRepository(db)
	showsRepo := dbLib.NewShowsPostgresRepository(db)
	bookingsRepo := dbLib.NewBookingsPostgresRepository(db)
	vipBundleRepo := dbLib.NewVipBundlePostgresRepository(db)
	bookingReadModel := dbLib.NewOpsBookingsReadModel(db, eventBus)
	dataLake := dbLib.NewDataLake(db)
	bookingsHandlers := event.NewOpsBookingHandlers(bookingReadModel)

	eventsHandler := event.NewHandler(
		eventBus,
		spreadsheetsService,
		receiptsService,
		fileService,
		deadNationService,
		paymentService,
		ticketsRepo,
		showsRepo,
	)

	commandBus, err := bus.NewCommandBus(redisPublisher)
	if err != nil {
		panic(fmt.Errorf("failed to create command bus: %w", err))
	}

	commandsHandler := command.NewHandler(
		eventBus,
		receiptsService,
		paymentService,
		transService,
		showsRepo,
		bookingsRepo,
	)

	postgresSubscriber := outbox.NewPostgresSubscriber(db.DB, watermillLogger)
	eventProcessorConfig := event.NewProcessorConfig(redisClient, watermillLogger)
	commandProcessorConfig := command.NewProcessorConfig(redisClient, watermillLogger)

	redisSubscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client: redisClient,
	}, watermillLogger)
	if err != nil {
		panic(fmt.Errorf("failed to create redis subscriber: %w", err))
	}

	vipBundleProcessManager := entity.NewVipBundleProcessManager(commandBus, eventBus, vipBundleRepo)
	watermillRouter, err := pubsub.NewWatermillRouter(
		postgresSubscriber,
		redisPublisher,
		redisSubscriber,
		eventProcessorConfig,
		eventsHandler,
		commandProcessorConfig,
		commandsHandler,
		bookingsHandlers,
		dataLake,
		vipBundleProcessManager,
		watermillLogger,
	)
	if err != nil {
		panic(fmt.Errorf("failed to create watermill router: %w", err))
	}

	httpServer := http.NewServer(
		addr,
		eventBus,
		commandBus,
		spreadsheetsService,
		ticketsRepo,
		showsRepo,
		bookingsRepo,
		bookingReadModel,
		vipBundleRepo,
	)

	return App{
		db,
		watermillRouter,
		httpServer,
		bookingsHandlers,
		dataLake,
		traceProvider,
	}
}

func (s App) Run(ctx context.Context) error {
	if err := dbLib.InitializeDatabaseSchema(s.db); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				veryImportantCounter.Inc()
				time.Sleep(time.Millisecond * 100)
			}
		}
	})

	g.Go(func() error {
		err := migrations.MigrateReadModel(ctx, s.dataLake, s.bookingHandlers)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to migrate read model: %s", err)
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		return s.traceProvider.Shutdown(context.Background())
	})

	g.Go(func() error {
		return s.watermillRouter.Run(ctx)
	})

	g.Go(func() error {
		// we don't want to start HTTP sferver before Watermill router (so app won't be healthy before it's ready)
		<-s.watermillRouter.Running()

		err := s.httpServer.Run(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return g.Wait()
}
