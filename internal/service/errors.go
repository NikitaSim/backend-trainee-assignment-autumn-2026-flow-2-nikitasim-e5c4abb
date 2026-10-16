package service

import "errors"

var (
	ErrCartEmpty             = errors.New("cart is empty")
	ErrRestaurantNotSelected = errors.New("restaurant is not selected")
	ErrRestaurantUnavailable = errors.New("restaurant is unavailable")
)
