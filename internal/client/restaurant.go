package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"avito-kitchen/internal/model"
)

type RestaurantClient interface {
	CreateOrder(ctx context.Context, order model.Order, items []model.OrderItem) (string, error)
}

type HTTPRestaurantClient struct {
	baseURL    string
	client     *http.Client
	maxRetries int
	backoff    time.Duration
}

func NewRestaurantClient(baseURL string) *HTTPRestaurantClient {
	return &HTTPRestaurantClient{
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 2 * time.Second},
		maxRetries: 3,
		backoff:    200 * time.Millisecond,
	}
}

func (c *HTTPRestaurantClient) CreateOrder(ctx context.Context, order model.Order, items []model.OrderItem) (string, error) {
	payload := struct {
		OrderID string `json:"order_id"`
		Items   []struct {
			MenuItemID string `json:"menu_item_id"`
			Quantity   int    `json:"quantity"`
		} `json:"items"`
	}{OrderID: order.ID.String()}

	for _, item := range items {
		payload.Items = append(payload.Items, struct {
			MenuItemID string `json:"menu_item_id"`
			Quantity   int    `json:"quantity"`
		}{item.MenuItemID.String(), item.Quantity})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		externalID, retry, err := c.createOrderAttempt(ctx, order.ID.String(), body)
		if err == nil {
			return externalID, nil
		}
		lastErr = err
		if !retry || attempt == c.maxRetries {
			break
		}

		delay := c.backoff * time.Duration(1<<attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}

	return "", fmt.Errorf("restaurant request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

func (c *HTTPRestaurantClient) createOrderAttempt(ctx context.Context, idempotencyKey string, body []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/orders", bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", false, err
		}
		return "", true, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", true, err
	}

	if resp.StatusCode >= 500 {
		return "", true, fmt.Errorf("restaurant returned status %d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("restaurant returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	var result struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", true, err
	}
	if result.OrderID == "" {
		return "", true, errors.New("restaurant returned empty order_id")
	}

	return result.OrderID, false, nil
}
