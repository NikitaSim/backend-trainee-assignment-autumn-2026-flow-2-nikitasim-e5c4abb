package service

import (
	"context"
	"errors"
	"testing"

	"avito-kitchen/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type orderRepoMock struct {
	created       bool
	existing      *model.Order
	updatedStatus string
}

func (m *orderRepoMock) Create(context.Context, model.Order, []model.OrderItem, string) (bool, error) {
	return m.created, nil
}
func (m *orderRepoMock) Get(context.Context, uuid.UUID) (model.Order, error) {
	return model.Order{}, nil
}
func (m *orderRepoMock) GetByIdempotencyKey(context.Context, uuid.UUID, string) (model.Order, error) {
	if m.existing == nil {
		return model.Order{}, pgx.ErrNoRows
	}
	return *m.existing, nil
}
func (m *orderRepoMock) List(context.Context, uuid.UUID) ([]model.Order, error)      { return nil, nil }
func (m *orderRepoMock) SetExternalOrderID(context.Context, uuid.UUID, string) error { return nil }
func (m *orderRepoMock) UpdateStatus(_ context.Context, _ uuid.UUID, status string) error {
	m.updatedStatus = status
	return nil
}

type cartRepoMock struct{ cart model.Cart }

func (m *cartRepoMock) Get(context.Context, uuid.UUID) (model.Cart, error)          { return m.cart, nil }
func (m *cartRepoMock) AddItem(context.Context, uuid.UUID, uuid.UUID, int) error    { return nil }
func (m *cartRepoMock) UpdateItem(context.Context, uuid.UUID, uuid.UUID, int) error { return nil }
func (m *cartRepoMock) DeleteItem(context.Context, uuid.UUID, uuid.UUID) error      { return nil }
func (m *cartRepoMock) Clear(context.Context, uuid.UUID) error                      { return nil }

type restaurantClientMock struct{ called bool }

func (m *restaurantClientMock) CreateOrder(context.Context, model.Order, []model.OrderItem) (string, error) {
	m.called = true
	return "external-1", nil
}

func TestOrderServiceCreateUsesIdempotencyResult(t *testing.T) {
	session := uuid.New()
	existing := model.Order{ID: uuid.New(), SessionID: session, Status: "confirmed"}
	repo := &orderRepoMock{existing: &existing}
	restaurant := &restaurantClientMock{}
	service := NewOrderService(repo, &cartRepoMock{}, restaurant)

	result, err := service.Create(context.Background(), session, "same-key")
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != existing.ID {
		t.Fatalf("expected existing order %s, got %s", existing.ID, result.ID)
	}
	if restaurant.called {
		t.Fatal("restaurant must not be called for an idempotent retry")
	}
}

func TestOrderServiceCreateRejectsEmptyCart(t *testing.T) {
	service := NewOrderService(&orderRepoMock{created: true}, &cartRepoMock{}, &restaurantClientMock{})
	_, err := service.Create(context.Background(), uuid.New(), "")
	if err == nil || err.Error() != "cart is empty" {
		t.Fatalf("expected cart is empty, got %v", err)
	}
}

func TestOrderServiceMarksOrderFailedWhenRestaurantUnavailable(t *testing.T) {
	session := uuid.New()
	repo := &orderRepoMock{created: true}
	restaurant := &restaurantClientErrorMock{}
	cart := &cartRepoMock{cart: model.Cart{SessionID: session, RestaurantID: uuid.New(), Items: []model.CartItem{{MenuItemID: uuid.New(), Name: "Pizza", Price: 500, Quantity: 1}}, TotalPrice: 500}}

	service := NewOrderService(repo, cart, restaurant)
	_, err := service.Create(context.Background(), session, "key")
	if !errors.Is(err, ErrRestaurantUnavailable) {
		t.Fatalf("expected restaurant unavailable, got %v", err)
	}
	if repo.updatedStatus != "failed" {
		t.Fatalf("expected failed status, got %q", repo.updatedStatus)
	}
}

type restaurantClientErrorMock struct{}

func (m *restaurantClientErrorMock) CreateOrder(context.Context, model.Order, []model.OrderItem) (string, error) {
	return "", errors.New("timeout")
}
