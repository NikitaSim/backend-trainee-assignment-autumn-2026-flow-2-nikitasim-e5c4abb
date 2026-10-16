package repository

import (
	"context"

	"avito-kitchen/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MenuRepository struct{ pool *pgxpool.Pool }

func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{pool: pool}
}

func (r *MenuRepository) List(ctx context.Context, restaurantID string) ([]model.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, restaurant_id, name, description, price, available
        FROM menu_items
        WHERE restaurant_id=$1
        ORDER BY name`, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.MenuItem
	for rows.Next() {
		var item model.MenuItem
		if err := rows.Scan(&item.ID, &item.RestaurantID, &item.Name, &item.Description, &item.Price, &item.Available); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
