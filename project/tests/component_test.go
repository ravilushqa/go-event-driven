package tests_test

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/alicebob/miniredis/v2"
	redis2 "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"

	httpHandler "tickets/handler/http"
	"tickets/handler/pubsub"
	"tickets/mocks"
	"tickets/pkg"
)

const (
	httpAddress = ":8080"
)

func TestComponent(t *testing.T) {
	defer goleak.VerifyNone(t)

	done := make(chan struct{})
	go func() {
		<-done
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.NoError(t, e)
	}()

	finished := make(chan struct{})
	go func() {
		err := startServer(t)
		assert.NoError(t, err)
		close(finished)
	}()

	defer func() {
		close(done)
		<-finished
	}()

	waitForHttpServer(t)
}

func startServer(t *testing.T) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	log.Init(logrus.InfoLevel)

	mr := miniredis.RunT(t)
	defer mr.Close()

	redisClient := pkg.NewRedisClient(mr.Addr())
	defer func(redisClient *redis2.Client) {
		err := redisClient.Close()
		assert.NoError(t, err)
	}(redisClient)

	receiptsClient := mocks.NewMockReceiptsService(t)
	spreadsheetsClient := mocks.NewMockSpreadsheetsAPI(t)
	watermillLogger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))

	redisPublisher := pkg.NewRedisPublisher(redisClient, watermillLogger)

	watermillRouter := pubsub.NewWatermillRouter(receiptsClient, spreadsheetsClient, redisClient, watermillLogger)

	httpServer := httpHandler.NewServer(redisPublisher, spreadsheetsClient, httpAddress)

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

	return g.Wait()
}

func waitForHttpServer(t *testing.T) {
	t.Helper()

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			resp, err := http.Get("http://localhost:8080/health")
			if !assert.NoError(t, err) {
				return
			}
			defer resp.Body.Close()

			if assert.Less(t, resp.StatusCode, 300, "API not ready, http status: %d", resp.StatusCode) {
				return
			}
		},
		time.Second*10,
		time.Millisecond*50,
	)
}
