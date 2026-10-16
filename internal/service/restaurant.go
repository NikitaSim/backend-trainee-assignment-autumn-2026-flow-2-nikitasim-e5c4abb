package service

import (
	"avito-kitchen/internal/model"
	"context"
)

type RestaurantRepository interface {
	List(context.Context) ([]model.Restaurant, error)
	Get(context.Context, string) (model.Restaurant, error)
}
type MenuRepository interface {
	List(context.Context, string) ([]model.MenuItem, error)
}

type RestaurantService struct {
	restaurants RestaurantRepository
	menu        MenuRepository
}

func NewRestaurantService(r RestaurantRepository, m MenuRepository) *RestaurantService {
	return &RestaurantService{restaurants: r, menu: m}
}

func (s *RestaurantService) List(ctx context.Context) ([]model.Restaurant, error) {
	return s.restaurants.List(ctx)
}
func (s *RestaurantService) Get(ctx context.Context, id string) (model.Restaurant, error) {
	return s.restaurants.Get(ctx, id)
}
func (s *RestaurantService) Menu(ctx context.Context, id string) ([]model.MenuItem, error) {
	return s.menu.List(ctx, id)
}
