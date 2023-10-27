package service

import (
	"context"

	"exercise/common"
	"exercise/orders"
)

func Run(ctx context.Context) {
	common.StartService(
		ctx,
		[]common.AddHandlersFn{
			orders.Initialize,
		},
	)
}
