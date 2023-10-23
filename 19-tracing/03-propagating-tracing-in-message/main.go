package main

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type PaymentReceived struct {
	ID            string `json:"id"`
	RoomBookingID string `json:"room_booking_id"`
}

type RoomBookingConfirmed struct {
	RoomBookingID string `json:"room_booking_id"`
}

type TraceDecorator struct {
	message.Publisher
}

func (d TraceDecorator) Publish(topic string, messages ...*message.Message) error {
	for i := range messages {
		otel.GetTextMapPropagator().Inject(messages[i].Context(), propagation.MapCarrier(messages[i].Metadata))
	}
	return d.Publisher.Publish(topic, messages...)
}

func NewRouter(rdb *redis.Client, logger watermill.LoggerAdapter) (*message.Router, *cqrs.EventBus) {
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}

	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) (events []*message.Message, err error) {
			// TODO: place for your middleware
			ctx := otel.GetTextMapPropagator().Extract(msg.Context(), propagation.MapCarrier(msg.Metadata))
			topic := message.SubscribeTopicFromCtx(msg.Context())
			handler := message.HandlerNameFromCtx(msg.Context())
			ctx, span := otel.Tracer("").Start(ctx, "message handling")
			span.SetAttributes(
				attribute.String("topic", topic),
				attribute.String("handler", handler),
			)
			defer span.End()
			msg.SetContext(ctx)

			messages, err := h(msg)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			return messages, err
		}
	})

	var pub message.Publisher
	pub, err = redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client: rdb,
		},
		logger,
	)
	if err != nil {
		panic(err)
	}

	// TODO: add tracing decorator
	pub = TraceDecorator{pub}

	marshaler := cqrs.JSONMarshaler{
		GenerateName: cqrs.StructName,
	}
	eventBus, err := cqrs.NewEventBusWithConfig(
		pub,
		cqrs.EventBusConfig{
			GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: marshaler,
		},
	)
	if err != nil {
		panic(err)
	}

	processor, err := newEventProcessor(router, rdb, marshaler, logger)
	if err != nil {
		panic(err)
	}

	err = processor.AddHandlers(cqrs.NewEventHandler(
		"PaymentReceived",
		func(ctx context.Context, event *PaymentReceived) error {
			return eventBus.Publish(ctx, RoomBookingConfirmed{
				RoomBookingID: event.RoomBookingID,
			})
		},
	))
	if err != nil {
		panic(err)
	}

	return router, eventBus
}

func initTracing(exp sdktrace.SpanExporter) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("ExampleService"),
		),
	)
	if err != nil {
		panic(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithResource(r),
	)

	otel.SetTracerProvider(tp)

	// todo: add propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})
}

func newEventProcessor(
	router *message.Router,
	rdb *redis.Client,
	marshaler cqrs.CommandEventMarshaler,
	logger watermill.LoggerAdapter,
) (*cqrs.EventProcessor, error) {
	return cqrs.NewEventProcessorWithConfig(
		router,
		cqrs.EventProcessorConfig{
			SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
				return redisstream.NewSubscriber(
					redisstream.SubscriberConfig{
						Client:        rdb,
						ConsumerGroup: "svc-something." + params.HandlerName,
					},
					logger,
				)
			},
			GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
				return params.EventName, nil
			},
			Marshaler: marshaler,
			Logger:    logger,
		},
	)
}
