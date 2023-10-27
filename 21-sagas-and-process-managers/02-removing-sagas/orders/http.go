package orders

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/lib/pq"

	"exercise/common"

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

func mountHttpHandlers(e *echo.Echo, db *sqlx.DB) {
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

				products := make([]uuid.UUID, 0, len(order.Products))
				for product := range order.Products {
					products = append(products, product)
				}

				type Stock struct {
					ProductID uuid.UUID `db:"product_id"`
					Quantity  int       `db:"quantity"`
				}
				stocks := make([]Stock, 0, len(order.Products))

				err = tx.SelectContext(
					ctx,
					&stocks,
					"SELECT product_id, quantity FROM stock WHERE product_id = ANY($1)",
					pq.Array(products),
				)
				if err != nil {
					return err
				}

				for _, stock := range stocks {
					if stock.Quantity < order.Products[stock.ProductID] {
						// Mark Order as Cancelled
						_, err = tx.Exec(
							"UPDATE orders SET cancelled = true WHERE order_id = $1",
							order.OrderID,
						)
						if err != nil {
							return err
						}

						return nil
					}
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

					// Reduce Product Quantity (Remove from Stock)
					_, err = tx.Exec(
						"UPDATE stock SET quantity = quantity - $1 WHERE product_id = $2",
						quantity,
						product,
					)
					if err != nil {
						return err
					}

					// Mark Order as Shipped
					_, err = tx.Exec(
						"UPDATE orders SET shipped = true WHERE order_id = $1",
						order.OrderID,
					)
					if err != nil {
						return err
					}
				}

				return nil
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

	e.POST("/products-stock", func(c echo.Context) error {
		productStock := common.ProductStock{}
		if err := c.Bind(&productStock); err != nil {
			return err
		}
		if productStock.Quantity <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "quantity must be greater than 0")
		}
		if productStock.ProductID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "product_id must be provided")
		}

		_, err := db.Exec(`
			INSERT INTO stock (product_id, quantity)
			VALUES ($1, $2)
			ON CONFLICT (product_id) DO UPDATE SET quantity = stock.quantity + $2
		`, productStock.ProductID, productStock.Quantity)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	})
}
