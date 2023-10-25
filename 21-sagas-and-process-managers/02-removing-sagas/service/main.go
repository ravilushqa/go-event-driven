package service

import (
	"context"
	"exercise/common"
	"exercise/orders"
	"exercise/stock"
)

func Run(ctx context.Context) {
	common.StartService(
		ctx,
		[]common.AddHandlersFn{
			orders.Initialize,
			stock.Initialize,
		},
	)
}
