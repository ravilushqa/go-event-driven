package common

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type AddHandlersFn func(
	echo *echo.Echo,
)

func StartService(ctx context.Context, addMessageHandlers []AddHandlersFn) {
	log.Init(logrus.InfoLevel)

	e := commonHTTP.NewEcho()

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	for _, addHandlerFn := range addMessageHandlers {
		addHandlerFn(e)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	errgrp := errgroup.Group{}

	errgrp.Go(func() error {
		return e.Start(":8080")
	})

	errgrp.Go(func() error {
		<-ctx.Done()
		return e.Shutdown(ctx)
	})

	if err := errgrp.Wait(); err != nil {
		panic(err)
	}
}
