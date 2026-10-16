package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"avito-kitchen/internal/client"
	"avito-kitchen/internal/config"
	"avito-kitchen/internal/handler"
	"avito-kitchen/internal/repository"
	"avito-kitchen/internal/service"
	"avito-kitchen/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	pool, err := storage.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := storage.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}

	restaurantRepo := repository.NewRestaurantRepository(pool)
	menuRepo := repository.NewMenuRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)

	restaurantClient := client.NewRestaurantClient(cfg.RestaurantURL)

	restaurantService := service.NewRestaurantService(restaurantRepo, menuRepo)
	cartService := service.NewCartService(cartRepo, menuRepo)
	orderService := service.NewOrderService(orderRepo, cartRepo, restaurantClient)

	h := handler.New(restaurantService, cartService, orderService)
	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           h.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("kitchen-api listening on :%s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("kitchen-api stopped: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("kitchen-api shutdown: %v", err)
	}
}
