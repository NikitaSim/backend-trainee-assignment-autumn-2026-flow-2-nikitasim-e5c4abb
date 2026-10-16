package handler

import (
	"avito-kitchen/internal/service"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Handler struct {
	restaurants *service.RestaurantService
	cart        *service.CartService
	orders      *service.OrderService
}

func New(r *service.RestaurantService, c *service.CartService, o *service.OrderService) *Handler {
	return &Handler{restaurants: r, cart: c, orders: o}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/v1/restaurants", h.listRestaurants)
	mux.HandleFunc("GET /api/v1/restaurants/{id}", h.getRestaurant)
	mux.HandleFunc("GET /api/v1/restaurants/{id}/menu", h.menu)
	mux.HandleFunc("GET /api/v1/cart", h.getCart)
	mux.HandleFunc("POST /api/v1/cart/items", h.addCartItem)
	mux.HandleFunc("PATCH /api/v1/cart/items/{id}", h.updateCartItem)
	mux.HandleFunc("DELETE /api/v1/cart/items/{id}", h.deleteCartItem)
	mux.HandleFunc("POST /api/v1/orders", h.createOrder)
	mux.HandleFunc("GET /api/v1/orders", h.listOrders)
	mux.HandleFunc("GET /api/v1/orders/{id}", h.getOrder)
	return Chain(mux, Recovering, logging)
}

func sessionID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.Header.Get("X-Session-ID"))
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listRestaurants(w http.ResponseWriter, r *http.Request) {
	result, err := h.restaurants.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getRestaurant(w http.ResponseWriter, r *http.Request) {
	result, err := h.restaurants.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "restaurant not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) menu(w http.ResponseWriter, r *http.Request) {
	result, err := h.restaurants.Menu(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getCart(w http.ResponseWriter, r *http.Request) {
	session, err := sessionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid X-Session-ID")
		return
	}
	result, err := h.cart.Get(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) addCartItem(w http.ResponseWriter, r *http.Request) {
	session, err := sessionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid X-Session-ID")
		return
	}
	var input struct {
		MenuItemID string `json:"menu_item_id"`
		Quantity   int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	item, err := uuid.Parse(input.MenuItemID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid menu_item_id")
		return
	}
	if err := h.cart.Add(r.Context(), session, item, input.Quantity); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.getCart(w, r)
}

func (h *Handler) updateCartItem(w http.ResponseWriter, r *http.Request) {
	session, err := sessionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid X-Session-ID")
		return
	}
	item, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	var input struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.cart.Update(r.Context(), session, item, input.Quantity); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.getCart(w, r)
}

func (h *Handler) deleteCartItem(w http.ResponseWriter, r *http.Request) {
	session, err := sessionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid X-Session-ID")
		return
	}
	item, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}
	if err := h.cart.Delete(r.Context(), session, item); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	session, err := sessionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid X-Session-ID")
		return
	}
	result, err := h.orders.Create(r.Context(), session, r.Header.Get("Idempotency-Key"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCartEmpty), errors.Is(err, service.ErrRestaurantNotSelected):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrRestaurantUnavailable):
			writeError(w, http.StatusBadGateway, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) listOrders(w http.ResponseWriter, r *http.Request) {
	session, err := sessionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid X-Session-ID")
		return
	}
	result, err := h.orders.List(r.Context(), session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	result, err := h.orders.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type ResponseWriter struct {
	Status int
	http.ResponseWriter
}

func (rw *ResponseWriter) WriteHeader(status int) {
	rw.Status = status
	rw.ResponseWriter.WriteHeader(status)
}

func Recovering(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				writeError(w, http.StatusInternalServerError, "server crash")
				return
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		rw := &ResponseWriter{
			Status:         http.StatusOK,
			ResponseWriter: w,
		}

		start := time.Now()

		next.ServeHTTP(rw, r)

		log.Println("Status", rw.Status)
		log.Println("Method", r.Method)
		log.Println("Path:", r.URL.Path)
		log.Println("Time", time.Since(start))
	})
}

func Chain(handler http.Handler, middlewares ...func(next http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
