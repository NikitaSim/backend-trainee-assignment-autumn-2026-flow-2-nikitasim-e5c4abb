package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"avito-kitchen/internal/model"

	"github.com/google/uuid"
)

func testOrder() (model.Order, []model.OrderItem) {
	return model.Order{ID: uuid.New()}, []model.OrderItem{{ID: uuid.New(), MenuItemID: uuid.New(), Quantity: 2}}
}

func TestHTTPRestaurantClientRetriesTemporaryErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if calls.Load() < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"order_id": "restaurant-1"})
	}))
	defer server.Close()

	client := NewRestaurantClient(server.URL)
	client.backoff = time.Millisecond
	order, items := testOrder()

	id, err := client.CreateOrder(context.Background(), order, items)
	if err != nil {
		t.Fatal(err)
	}
	if id != "restaurant-1" {
		t.Fatalf("unexpected external id: %s", id)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

func TestHTTPRestaurantClientDoesNotRetryBadRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid order"}`))
	}))
	defer server.Close()

	client := NewRestaurantClient(server.URL)
	client.backoff = time.Millisecond
	order, items := testOrder()

	_, err := client.CreateOrder(context.Background(), order, items)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", calls.Load())
	}
}

func TestHTTPRestaurantClientStopsOnContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
	}))

	client := NewRestaurantClient(server.URL)

	order, items := testOrder()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.CreateOrder(ctx, order, items)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}

	server.CloseClientConnections()
	server.Close()
}
