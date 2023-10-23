package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MessagesProcessed The total number of processed messages (counter)
	MessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "messages",
			Name:      "processed_total",
			Help:      "The total number of processed messages",
		},
		[]string{"topic", "handler"},
	)

	// MessagesProcessingFailed total number of message processing failures (counter)
	MessagesProcessingFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "messages",
			Name:      "processing_failed_total",
			Help:      "The total number of message processing failures",
		},
		[]string{"topic", "handler"},
	)

	// MessagesProcessingDuration The total time spent processing messages (summary with quantiles 0.5, 0.9, and 0.99)
	MessagesProcessingDuration = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "messages",
			Name:       "processing_duration_seconds",
			Help:       "The total time spent processing messages",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"topic", "handler"},
	)
)
