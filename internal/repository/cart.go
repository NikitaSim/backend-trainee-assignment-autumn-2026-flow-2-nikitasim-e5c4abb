package repository

import (
	"context"
	"errors"

	"avito-kitchen/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CartRepository struct{ pool *pgxpool.Pool }

func NewCartRepository(pool *pgxpool.Pool) *CartRepository {
	return &CartRepository{pool: pool}
}

func (r *CartRepository) Get(ctx context.Context, sessionID uuid.UUID) (model.Cart, error) {
	var cart model.Cart
	err := r.pool.QueryRow(ctx, `
        INSERT INTO carts(session_id)
        VALUES($1)
        ON CONFLICT(session_id) DO UPDATE SET updated_at=now()
        RETURNING id, session_id, restaurant_id`, sessionID).
		Scan(&cart.ID, &cart.SessionID, &cart.RestaurantID)
	if err != nil {
		return cart, err
	}

	rows, err := r.pool.Query(ctx, `
        SELECT ci.id, ci.menu_item_id, mi.name, ci.quantity, mi.price
        FROM cart_items ci
        JOIN menu_items mi ON mi.id=ci.menu_item_id
        WHERE ci.cart_id=$1
        ORDER BY mi.name`, cart.ID)
	if err != nil {
		return cart, err
	}
	defer rows.Close()

	for rows.Next() {
		var item model.CartItem
		if err := rows.Scan(&item.ID, &item.MenuItemID, &item.Name, &item.Quantity, &item.Price); err != nil {
			return cart, err
		}
		cart.Items = append(cart.Items, item)
		cart.TotalPrice += item.Price * int64(item.Quantity)
	}
	return cart, rows.Err()
}

func (r *CartRepository) AddItem(ctx context.Context, sessionID, menuItemID uuid.UUID, quantity int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var cartID uuid.UUID
	var currentRestaurant *uuid.UUID
	err = tx.QueryRow(ctx, `
        INSERT INTO carts(session_id)
        VALUES($1)
        ON CONFLICT(session_id) DO UPDATE SET updated_at=now()
        RETURNING id, restaurant_id`, sessionID).Scan(&cartID, &currentRestaurant)
	if err != nil {
		return err
	}

	var restaurantID uuid.UUID
	var available bool
	err = tx.QueryRow(ctx, `SELECT restaurant_id, available FROM menu_items WHERE id=$1`, menuItemID).
		Scan(&restaurantID, &available)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("menu item not found")
	}
	if err != nil {
		return err
	}
	if !available {
		return errors.New("menu item is unavailable")
	}

	if currentRestaurant != nil && *currentRestaurant != restaurantID {
		return errors.New("cart can contain items from one restaurant only")
	}

	_, err = tx.Exec(ctx, `UPDATE carts SET restaurant_id=$2, updated_at=now() WHERE id=$1`, cartID, restaurantID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO cart_items(cart_id, menu_item_id, quantity)
        VALUES($1,$2,$3)
        ON CONFLICT(cart_id, menu_item_id)
        DO UPDATE SET quantity=cart_items.quantity + EXCLUDED.quantity, updated_at=now()`,
		cartID, menuItemID, quantity)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *CartRepository) UpdateItem(ctx context.Context, sessionID, itemID uuid.UUID, quantity int) error {
	result, err := r.pool.Exec(ctx, `
        UPDATE cart_items ci
        SET quantity=$1, updated_at=now()
        FROM carts c
        WHERE ci.id=$2 AND ci.cart_id=c.id AND c.session_id=$3`, quantity, itemID, sessionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("cart item not found")
	}
	return nil
}

func (r *CartRepository) DeleteItem(ctx context.Context, sessionID, itemID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `
        DELETE FROM cart_items ci
        USING carts c
        WHERE ci.id=$1 AND ci.cart_id=c.id AND c.session_id=$2`, itemID, sessionID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("cart item not found")
	}
	return nil
}

func (r *CartRepository) Clear(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
        DELETE FROM cart_items ci
        USING carts c
        WHERE ci.cart_id=c.id AND c.session_id=$1`, sessionID)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE carts SET restaurant_id=NULL, updated_at=now() WHERE session_id=$1`, sessionID)
	return err
}
