package pubsub

import (
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"tickets/metrics"
)

func useMiddlewares(router *message.Router, watermillLogger watermill.LoggerAdapter) {
	router.AddMiddleware(middleware.Recoverer)

	router.AddMiddleware(middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          watermillLogger,
	}.Middleware)

	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) (events []*message.Message, err error) {
			ctx := otel.GetTextMapPropagator().Extract(msg.Context(), propagation.MapCarrier(msg.Metadata))
			topic := message.SubscribeTopicFromCtx(msg.Context())
			handler := message.HandlerNameFromCtx(msg.Context())
			ctx, span := otel.Tracer("").Start(ctx, "message handling: "+topic+"/"+handler)
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

	router.AddMiddleware(func(next message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			traceID := trace.SpanFromContext(msg.Context()).SpanContext().TraceID().String()
			logger := log.FromContext(msg.Context()).WithFields(logrus.Fields{
				"message_id": msg.UUID,
				"payload":    string(msg.Payload),
				"metadata":   msg.Metadata,
				"trace_id":   traceID,
			})

			logger.Info("Handling a message")

			msgs, err := next(msg)
			if err != nil {
				logger.WithError(err).Error("Error while handling a message")
			}

			return msgs, err
		}
	})

	router.AddMiddleware(func(next message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			now := time.Now()
			topic := message.SubscribeTopicFromCtx(msg.Context())
			handler := message.HandlerNameFromCtx(msg.Context())
			labels := prometheus.Labels{"topic": topic, "handler": handler}
			var err error
			defer func() {
				if err != nil {
					metrics.MessagesProcessingFailed.With(labels).Inc()
				}
				metrics.MessagesProcessed.With(labels).Inc()
				metrics.MessagesProcessingDuration.With(labels).Observe(time.Since(now).Seconds())

			}()

			msgs, err := next(msg)
			return msgs, err
		}
	})
}
