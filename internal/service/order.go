package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"avito-kitchen/internal/client"
	"avito-kitchen/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OrderRepository interface {
	Create(context.Context, model.Order, []model.OrderItem, string) (bool, error)
	Get(context.Context, uuid.UUID) (model.Order, error)
	GetByIdempotencyKey(context.Context, uuid.UUID, string) (model.Order, error)
	List(context.Context, uuid.UUID) ([]model.Order, error)
	SetExternalOrderID(context.Context, uuid.UUID, string) error
	UpdateStatus(context.Context, uuid.UUID, string) error
}

type OrderService struct {
	orders     OrderRepository
	carts      CartRepository
	restaurant client.RestaurantClient
}

func NewOrderService(o OrderRepository, c CartRepository, r client.RestaurantClient) *OrderService {
	return &OrderService{orders: o, carts: c, restaurant: r}
}

func (s *OrderService) Create(ctx context.Context, session uuid.UUID, idempotencyKey string) (model.Order, error) {
	if idempotencyKey != "" {
		existing, err := s.orders.GetByIdempotencyKey(ctx, session, idempotencyKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.Order{}, err
		}
	}

	cart, err := s.carts.Get(ctx, session)
	if err != nil {
		return model.Order{}, err
	}
	if len(cart.Items) == 0 {
		return model.Order{}, ErrCartEmpty
	}
	if cart.RestaurantID == uuid.Nil {
		return model.Order{}, ErrRestaurantNotSelected
	}

	order := model.Order{
		ID: uuid.New(), SessionID: session, RestaurantID: cart.RestaurantID,
		Status: "pending", TotalPrice: cart.TotalPrice, CreatedAt: time.Now(),
	}
	items := make([]model.OrderItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		items = append(items, model.OrderItem{ID: uuid.New(), OrderID: order.ID, MenuItemID: item.MenuItemID, Name: item.Name, Price: item.Price, Quantity: item.Quantity})
	}

	created, err := s.orders.Create(ctx, order, items, idempotencyKey)
	if err != nil {
		return model.Order{}, err
	}
	if !created && idempotencyKey != "" {
		return s.orders.GetByIdempotencyKey(ctx, session, idempotencyKey)
	}

	externalID, err := s.restaurant.CreateOrder(ctx, order, items)
	if err != nil {
		_ = s.orders.UpdateStatus(ctx, order.ID, "failed")
		return order, fmt.Errorf("%w: %v", ErrRestaurantUnavailable, err)
	}

	if err := s.orders.SetExternalOrderID(ctx, order.ID, externalID); err != nil {
		return order, err
	}
	order.ExternalOrderID = &externalID
	order.Status = "confirmed"
	if err := s.carts.Clear(ctx, session); err != nil {
		return order, err
	}
	return order, nil
}

func (s *OrderService) Get(ctx context.Context, id uuid.UUID) (model.Order, error) {
	return s.orders.Get(ctx, id)
}
func (s *OrderService) List(ctx context.Context, session uuid.UUID) ([]model.Order, error) {
	return s.orders.List(ctx, session)
}
