package orders

import (
	"context"
	"database/sql"
	"exercise/common"
	"net/http"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type PostOrderRequest struct {
	OrderID   uuid.UUID         `json:"order_id"`
	Products  map[uuid.UUID]int `json:"products"`
	Shipped   bool              `json:"shipped"`
	Cancelled bool              `json:"cancelled"`
}

type GetOrderResponse struct {
	OrderID   uuid.UUID `json:"order_id" db:"order_id"`
	Shipped   bool      `json:"shipped" db:"shipped"`
	Cancelled bool      `json:"cancelled" db:"cancelled"`
}

func mountHttpHandlers(e *echo.Echo, db *sqlx.DB, eventBus *cqrs.EventBus) {
	e.POST("/orders", func(c echo.Context) error {
		order := PostOrderRequest{}
		if err := c.Bind(&order); err != nil {
			return err
		}

		err := common.UpdateInTx(
			c.Request().Context(),
			db,
			sql.LevelSerializable,
			func(ctx context.Context, tx *sqlx.Tx) error {
				_, err := tx.Exec(
					"INSERT INTO orders (order_id, shipped, cancelled) VALUES ($1, $2, $3)",
					order.OrderID,
					false,
					false,
				)
				if err != nil {
					return err
				}

				for product, quantity := range order.Products {
					_, err := tx.Exec(
						"INSERT INTO order_products (order_id, product_id, quantity) VALUES ($1, $2, $3)",
						order.OrderID,
						product,
						quantity,
					)
					if err != nil {
						return err
					}
				}

				// we should use outbox here, but I don't want to add you more stuff to remove
				return eventBus.Publish(ctx, &common.OrderPlaced{
					OrderID:  order.OrderID,
					Products: order.Products,
				})
			},
		)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	})

	e.GET("/orders/:order_id", func(c echo.Context) error {
		orderID, err := uuid.Parse(c.Param("order_id"))
		if err != nil {
			return err
		}

		order := GetOrderResponse{}

		err = db.Get(
			&order,
			"SELECT order_id, shipped, cancelled FROM orders WHERE order_id = $1",
			orderID,
		)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, PostOrderRequest{
			OrderID:   order.OrderID,
			Shipped:   order.Shipped,
			Cancelled: order.Cancelled,
		})
	})
}
