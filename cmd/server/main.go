package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"marketplace-backend/internal/middleware"
	"marketplace-backend/internal/operations"
	"marketplace-backend/internal/orders"
	"marketplace-backend/internal/products"
	"marketplace-backend/internal/promos"
	"marketplace-backend/internal/users"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

func connectWithRetry(dsn string, maxRetries int, retryDelay time.Duration) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			return db, nil
		}

		log.Printf("Failed to connect to database (attempt %d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Support both DATABASE_URL and individual DB_ variables
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		// Use individual variables
		dbHost := getEnv("DB_HOST", "localhost")
		dbPort := getEnv("DB_PORT", "5432")
		dbUser := getEnv("DB_USER", "marketplace")
		dbPassword := getEnv("DB_PASSWORD", "marketplace_password")
		dbName := getEnv("DB_NAME", "marketplace")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	} else {
		// DATABASE_URL is set - parse it to key=value format for lib/pq
		// lib/pq doesn't handle postgres:// URLs well, convert to connection string format
		// Expected format: postgres://user:pass@host:port/dbname?sslmode=disable
		// Convert to: host=host port=port user=user password=pass dbname=dbname sslmode=disable

		if len(dsn) > 11 && dsn[:11] == "postgres://" {
			// Remove postgres:// prefix
			rest := dsn[11:]

			// Split by @
			parts := splitOnce(rest, "@")
			if len(parts) == 2 {
				userPass := parts[0]
				hostPortDb := parts[1]

				// Split user:pass
				userPassParts := splitOnce(userPass, ":")
				user := userPassParts[0]
				password := ""
				if len(userPassParts) == 2 {
					password = userPassParts[1]
				}

				// Split host:port/db?params
				hostPortDbParts := splitOnce(hostPortDb, "/")
				hostPort := hostPortDbParts[0]
				dbParams := ""
				if len(hostPortDbParts) == 2 {
					dbParams = hostPortDbParts[1]
				}

				// Split host:port
				hostPortParts := splitOnce(hostPort, ":")
				host := hostPortParts[0]
				port := "5432"
				if len(hostPortParts) == 2 {
					port = hostPortParts[1]
				}

				// Split db?params
				dbParamsParts := splitOnce(dbParams, "?")
				dbname := dbParamsParts[0]

				// Reconstruct as key=value format
				dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
					host, port, user, password, dbname)
			}
		}
	}
	logger.Info().Str("dsn", maskPassword(dsn)).Msg("Connecting to database")

	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")

	// Retry connection up to 10 times with 2 second delay
	db, err := connectWithRetry(dsn, 10, 2*time.Second)
	if err != nil {
		logger.Error().Err(err).Str("dsn", maskPassword(dsn)).Msg("Failed to connect to database")
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	logger.Info().Msg("Database connection established")

	usersRepo := users.NewRepository(db)
	usersService := users.NewService(usersRepo, jwtSecret, 30*time.Minute, 7*24*time.Hour)
	usersHandler := users.NewHandler(usersService)

	productsRepo := products.NewRepository(db)
	productsService := products.NewService(productsRepo)
	productsHandler := products.NewHandler(productsService)

	promosRepo := promos.NewRepository(db)
	promosService := promos.NewService(promosRepo)
	promosHandler := promos.NewHandler(promosService)

	operationsRepo := operations.NewRepository(db)

	ordersRepo := orders.NewRepository(db)
	ordersService := orders.NewService(ordersRepo, productsRepo, promosRepo, operationsRepo, 5)
	ordersHandler := orders.NewHandler(ordersService)

	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.LoggingMiddleware)

	r.Post("/auth/register", usersHandler.Register)
	r.Post("/auth/login", usersHandler.Login)
	r.Post("/auth/refresh", usersHandler.Refresh)

	r.Route("/products", func(r chi.Router) {
		r.Get("/", productsHandler.ListProducts)
		r.Get("/{id}", productsHandler.GetProduct)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(usersService))
			r.Post("/", productsHandler.CreateProduct)
			r.Put("/{id}", productsHandler.UpdateProduct)
			r.Delete("/{id}", productsHandler.DeleteProduct)
		})
	})

	r.Route("/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(usersService))
		r.Post("/", ordersHandler.CreateOrder)
		r.Get("/{id}", ordersHandler.GetOrder)
		r.Put("/{id}", ordersHandler.UpdateOrder)
		r.Post("/{id}/cancel", ordersHandler.CancelOrder)
	})

	r.Route("/promo-codes", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(usersService))
		r.Post("/", promosHandler.CreatePromoCode)
		r.Get("/{id}", promosHandler.GetPromoCode)
		r.Put("/{id}", promosHandler.UpdatePromoCode)
		r.Delete("/{id}", promosHandler.DeletePromoCode)
	})

	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	logger.Info().Str("addr", addr).Msg("Starting server")

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskPassword(dsn string) string {
	// Simple password masking for logging
	if len(dsn) > 20 {
		return dsn[:20] + "...***..."
	}
	return "***"
}

func splitOnce(s, sep string) []string {
	idx := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			idx = i
			return []string{s[:idx], s[idx+len(sep):]}
		}
	}
	return []string{s}
}
