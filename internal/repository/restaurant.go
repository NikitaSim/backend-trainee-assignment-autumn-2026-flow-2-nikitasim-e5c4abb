package repository

import (
	"context"

	"avito-kitchen/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RestaurantRepository struct{ pool *pgxpool.Pool }

func NewRestaurantRepository(pool *pgxpool.Pool) *RestaurantRepository {
	return &RestaurantRepository{pool: pool}
}

func (r *RestaurantRepository) List(ctx context.Context) ([]model.Restaurant, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, description, address FROM restaurants ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Restaurant
	for rows.Next() {
		var item model.Restaurant
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Address); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *RestaurantRepository) Get(ctx context.Context, id string) (model.Restaurant, error) {
	var item model.Restaurant
	err := r.pool.QueryRow(ctx, `SELECT id, name, description, address FROM restaurants WHERE id=$1`, id).
		Scan(&item.ID, &item.Name, &item.Description, &item.Address)
	return item, err
}
