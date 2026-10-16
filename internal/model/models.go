package model

import (
	"time"

	"github.com/google/uuid"
)

type Restaurant struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Address     string    `json:"address"`
}

type MenuItem struct {
	ID           uuid.UUID `json:"id"`
	RestaurantID uuid.UUID `json:"restaurant_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        int64     `json:"price"`
	Available    bool      `json:"available"`
}

type CartItem struct {
	ID         uuid.UUID `json:"id"`
	MenuItemID uuid.UUID `json:"menu_item_id"`
	Name       string    `json:"name"`
	Price      int64     `json:"price"`
	Quantity   int       `json:"quantity"`
}

type Cart struct {
	ID           uuid.UUID  `json:"id"`
	SessionID    uuid.UUID  `json:"session_id"`
	RestaurantID uuid.UUID  `json:"restaurant_id"`
	Items        []CartItem `json:"items"`
	TotalPrice   int64      `json:"total_price"`
}

type Order struct {
	ID              uuid.UUID `json:"id"`
	SessionID       uuid.UUID `json:"session_id"`
	RestaurantID    uuid.UUID `json:"restaurant_id"`
	Status          string    `json:"status"`
	TotalPrice      int64     `json:"total_price"`
	ExternalOrderID *string   `json:"external_order_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type OrderItem struct {
	ID         uuid.UUID `json:"id"`
	OrderID    uuid.UUID `json:"order_id"`
	MenuItemID uuid.UUID `json:"menu_item_id"`
	Name       string    `json:"name"`
	Price      int64     `json:"price"`
	Quantity   int       `json:"quantity"`
}
