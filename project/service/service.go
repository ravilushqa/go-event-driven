package service

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	dbLib "tickets/db"
	"tickets/db/bookings"
	dl "tickets/db/data_lake"
	"tickets/db/read_model_ops_bookings"
	"tickets/db/shows"
	"tickets/db/tickets"
	"tickets/http"
	migrations "tickets/migration"
	"tickets/pubsub"
	"tickets/pubsub/bus"
	"tickets/pubsub/command"
	"tickets/pubsub/event"
	"tickets/pubsub/outbox"
)

func init() {
	log.Init(logrus.InfoLevel)
}

type Service struct {
	db              *sqlx.DB
	watermillRouter *message.Router
	httpServer      *http.Server
	opsReadModel    read_model_ops_bookings.OpsBookingReadModel
	dataLake        dl.DataLake
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
) Service {
	var redisPublisher message.Publisher

	watermillLogger := log.NewWatermill(log.FromContext(context.Background()))
	redisPublisher = pubsub.NewRedisPublisher(redisClient, watermillLogger)
	redisPublisher = log.CorrelationPublisherDecorator{Publisher: redisPublisher}

	eventBus, err := bus.NewEventBus(redisPublisher)
	if err != nil {
		panic(fmt.Errorf("failed to create event bus: %w", err))
	}

	ticketsRepo := tickets.NewPostgresRepository(db)
	showsRepo := shows.NewPostgresRepository(db)
	bookingsRepo := bookings.NewPostgresRepository(db)
	opsReadModel := read_model_ops_bookings.NewOpsBookingReadModel(db, eventBus)

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

	dataLake := dl.NewDataLake(db)
	watermillRouter, err := pubsub.NewWatermillRouter(
		postgresSubscriber,
		redisPublisher,
		redisSubscriber,
		eventProcessorConfig,
		eventsHandler,
		commandProcessorConfig,
		commandsHandler,
		opsReadModel,
		dataLake,
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
		opsReadModel,
	)

	return Service{
		db,
		watermillRouter,
		httpServer,
		opsReadModel,
		dataLake,
	}
}

func (s Service) Run(ctx context.Context) error {
	if err := dbLib.InitializeDatabaseSchema(s.db); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := migrations.MigrateReadModel(ctx, s.dataLake, s.opsReadModel)
		if err != nil {
			log.FromContext(ctx).Errorf("failed to migrate read model: %s", err)
		}
		return nil
	})

	g.Go(func() error {
		return s.watermillRouter.Run(ctx)
	})

	g.Go(func() error {
		// we don't want to start HTTP sferver before Watermill router (so service won't be healthy before it's ready)
		<-s.watermillRouter.Running()

		err := s.httpServer.Run(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return g.Wait()
}

//func migrateDB(ctx context.Context, dbConn *sqlx.DB, opsReadModel read_model_ops_bookings.OpsBookingReadModel) error {
//	err := db.InitializeDatabaseSchema(dbConn)
//	if err != nil {
//		return err
//	}
//	l := log.FromContext(ctx)
//
//	var events []entity.DataLakeEvent
//	for {
//		l.Info("Waiting for events table to be created")
//		err = dbConn.Select(&events, "SELECT * FROM events")
//		if err != nil {
//			return err
//		}
//
//		if len(events) != 0 {
//			l.Infof("Events table created, %d events found", len(events))
//			break
//		}
//
//		time.Sleep(1 * time.Second)
//	}
//
//	for _, e := range events {
//		l.Infof("Migrating event %s(%s)", e.Name, e.ID)
//		l.Debugf("Event payload: %s", string(e.Payload))
//		if strings.HasPrefix(e.Name, "BookingMade") {
//			eventModel := entity.BookingMade_v1{}
//			err := json.Unmarshal(e.Payload, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to unmarshal event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//
//			err = opsReadModel.OnBookingMade(ctx, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to migrate event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//		}
//
//		if strings.HasPrefix(e.Name, "TicketBookingConfirmed") {
//			eventModel := entity.TicketBookingConfirmed_v1{}
//			err := json.Unmarshal(e.Payload, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to unmarshal event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//
//			err = opsReadModel.OnTicketBookingConfirmed(ctx, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to migrate event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//		}
//
//		if strings.HasPrefix(e.Name, "TicketReceiptIssued") {
//			eventModel := entity.TicketReceiptIssued_v1{}
//			err := json.Unmarshal(e.Payload, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to unmarshal event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//
//			err = opsReadModel.OnTicketReceiptIssued(ctx, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to migrate event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//		}
//
//		if strings.HasPrefix(e.Name, "TicketRefunded") {
//			eventModel := entity.TicketRefunded_v1{}
//			err := json.Unmarshal(e.Payload, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to unmarshal event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//
//			err = opsReadModel.OnTicketRefunded(ctx, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to migrate event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//		}
//
//		if strings.HasPrefix(e.Name, "TicketPrinted") {
//			eventModel := entity.TicketPrinted_v1{}
//			err := json.Unmarshal(e.Payload, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to unmarshal event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//
//			err = opsReadModel.OnTicketPrinted(ctx, &eventModel)
//			if err != nil {
//				l.Errorf("Failed to migrate event %s(%s): %s", e.Name, e.ID, err)
//				continue
//			}
//		}
//	}
//
//	return nil
//}
