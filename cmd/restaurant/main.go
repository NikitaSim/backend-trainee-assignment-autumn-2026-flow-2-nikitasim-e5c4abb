package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type order struct {
	ID        string      `json:"order_id"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	Items     []orderItem `json:"items"`
}

type orderItem struct {
	MenuItemID string `json:"menu_item_id"`
	Quantity   int    `json:"quantity"`
}

type createOrderRequest struct {
	OrderID string      `json:"order_id"`
	Items   []orderItem `json:"items"`
}

type mockServer struct {
	mu       sync.RWMutex
	orders   map[string]order
	failLeft int
	delay    time.Duration
	failCode int
}

func main() {
	port := getenv("HTTP_PORT", "8081")
	mock := &mockServer{
		orders:   make(map[string]order),
		failLeft: getenvInt("MOCK_FAIL_COUNT", 0),
		delay:    time.Duration(getenvInt("MOCK_DELAY_MS", 0)) * time.Millisecond,
		failCode: getenvInt("MOCK_FAIL_STATUS", http.StatusServiceUnavailable),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", mock.health)
	mux.HandleFunc("GET /api/v1/menu", mock.menu)
	mux.HandleFunc("POST /api/v1/orders", mock.createOrder)
	mux.HandleFunc("GET /api/v1/orders/{id}", mock.getOrder)
	mux.HandleFunc("PATCH /api/v1/orders/{id}/status", mock.updateStatus)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("restaurant-mock listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("restaurant-mock stopped: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("restaurant-mock shutdown: %v", err)
	}
}

func (s *mockServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *mockServer) menu(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []map[string]any{
			{"menu_item_id": uuid.New().String(), "name": "Margherita", "price": 65000},
			{"menu_item_id": uuid.New().String(), "name": "Pepperoni", "price": 79000},
		},
	})
}

func (s *mockServer) createOrder(w http.ResponseWriter, r *http.Request) {
	if s.delay > 0 {
		timer := time.NewTimer(s.delay)
		select {
		case <-r.Context().Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}

	var input createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid order"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.orders[input.OrderID]; ok {
		writeJSON(w, http.StatusOK, existing)
		return
	}

	if s.failLeft > 0 {
		s.failLeft--
		writeJSON(w, s.failCode, map[string]string{"error": "temporary restaurant failure"})
		return
	}

	created := order{
		ID:        "restaurant-" + uuid.NewString(),
		Status:    "accepted",
		CreatedAt: time.Now(),
		Items:     input.Items,
	}
	s.orders[input.OrderID] = created
	writeJSON(w, http.StatusAccepted, created)
}

func (s *mockServer) getOrder(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.orders {
		if item.ID == r.PathValue("id") {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
}

func (s *mockServer) updateStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if !validStatus(input.Status) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for key, item := range s.orders {
		if item.ID == r.PathValue("id") {
			if !canTransition(item.Status, input.Status) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid status transition"})
				return
			}
			item.Status = input.Status
			s.orders[key] = item
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
}

func validStatus(status string) bool {
	switch status {
	case "accepted", "preparing", "ready", "delivered", "cancelled":
		return true
	default:
		return false
	}
}

func canTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case "accepted":
		return to == "preparing" || to == "cancelled"
	case "preparing":
		return to == "ready" || to == "cancelled"
	case "ready":
		return to == "delivered" || to == "cancelled"
	case "delivered", "cancelled":
		return false
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid %s=%q, using %d: %v", key, value, fallback, err)
		return fallback
	}
	return parsed
}
