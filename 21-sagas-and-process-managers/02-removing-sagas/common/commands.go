package common

import "github.com/google/uuid"

type CancelOrder struct {
	OrderID uuid.UUID `json:"order_id"`
}

type RemoveProductsFromStock struct {
	OrderID  uuid.UUID         `json:"order_id"`
	Products map[uuid.UUID]int `json:"products"`
}

type ShipOrder struct {
	OrderID uuid.UUID `json:"order_id"`
}

type ProductStock struct {
	ProductID string `db:"product_id" json:"product_id"`
	Quantity  int    `db:"quantity" json:"quantity"`
}
