package repository

import (
	"context"
	"errors"

	"avito-kitchen/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct{ pool *pgxpool.Pool }

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, order model.Order, items []model.OrderItem, idempotencyKey string) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var inserted bool
	err = tx.QueryRow(ctx, `
        INSERT INTO orders(id, session_id, restaurant_id, status, total_price, external_order_id, idempotency_key)
        VALUES($1,$2,$3,$4,$5,$6,$7)
        ON CONFLICT(session_id, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
        RETURNING true`, order.ID, order.SessionID, order.RestaurantID, order.Status, order.TotalPrice, order.ExternalOrderID, idempotencyKey).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	for _, item := range items {
		if _, err := tx.Exec(ctx, `
            INSERT INTO order_items(id, order_id, menu_item_id, name, price, quantity)
            VALUES($1,$2,$3,$4,$5,$6)`, item.ID, order.ID, item.MenuItemID, item.Name, item.Price, item.Quantity); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (r *OrderRepository) Get(ctx context.Context, id uuid.UUID) (model.Order, error) {
	var order model.Order
	err := r.pool.QueryRow(ctx, `
        SELECT id, session_id, restaurant_id, status, total_price, external_order_id, created_at
        FROM orders WHERE id=$1`, id).
		Scan(&order.ID, &order.SessionID, &order.RestaurantID, &order.Status, &order.TotalPrice, &order.ExternalOrderID, &order.CreatedAt)
	return order, err
}

func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, sessionID uuid.UUID, key string) (model.Order, error) {
	var order model.Order
	err := r.pool.QueryRow(ctx, `
        SELECT id, session_id, restaurant_id, status, total_price, external_order_id, created_at
        FROM orders WHERE session_id=$1 AND idempotency_key=$2`, sessionID, key).
		Scan(&order.ID, &order.SessionID, &order.RestaurantID, &order.Status, &order.TotalPrice, &order.ExternalOrderID, &order.CreatedAt)
	return order, err
}

func (r *OrderRepository) SetExternalOrderID(ctx context.Context, id uuid.UUID, externalID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE orders SET external_order_id=$2, status='confirmed', updated_at=now() WHERE id=$1`, id, externalID)
	return err
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *OrderRepository) List(ctx context.Context, sessionID uuid.UUID) ([]model.Order, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, session_id, restaurant_id, status, total_price, external_order_id, created_at
        FROM orders WHERE session_id=$1 ORDER BY created_at DESC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Order
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.ID, &order.SessionID, &order.RestaurantID, &order.Status, &order.TotalPrice, &order.ExternalOrderID, &order.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, order)
	}
	return result, rows.Err()
}
