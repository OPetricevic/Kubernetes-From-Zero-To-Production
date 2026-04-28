package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

const cacheKey = "products:all"
const cacheTTL = 30 * time.Second

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrations()
		return
	}

	connectDB()
	defer db.Close()
	connectRedis()
	defer rdb.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/products", listProductsHandler)
	mux.HandleFunc("GET /api/products/{id}", getProductHandler)
	mux.HandleFunc("POST /api/products", createProductHandler)
	mux.HandleFunc("PUT /api/products/{id}", updateProductHandler)
	mux.HandleFunc("DELETE /api/products/{id}", deleteProductHandler)

	port := getEnv("PORT", "8081")
	log.Printf("Product service starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func connectDB() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "shopstream"),
		getEnv("DB_PASSWORD", "shopstream123"),
		getEnv("DB_NAME", "shopstream"),
	)

	var err error
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Println("Connected to database")
				return
			}
		}
		log.Printf("Waiting for database... attempt %d/30", i+1)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("Could not connect to database: %v", err)
}

func connectRedis() {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	for i := 0; i < 10; i++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			log.Println("Connected to Redis")
			return
		}
		log.Printf("Waiting for Redis... attempt %d/10", i+1)
		time.Sleep(2 * time.Second)
	}
	log.Println("Redis not available — running without cache")
}

func runMigrations() {
	connectDB()
	defer db.Close()

	migration := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE TABLE IF NOT EXISTS products (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(255) NOT NULL,
		description TEXT DEFAULT '',
		price DECIMAL(10,2) NOT NULL DEFAULT 0,
		stock INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	_, err := db.Exec(migration)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migrations completed successfully")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// listProductsHandler tries Redis cache first, falls back to DB
func listProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Try cache first
	cached, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		log.Println("Cache HIT — serving from Redis")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write([]byte(cached))
		return
	}

	log.Println("Cache MISS — querying database")
	rows, err := db.Query("SELECT id, name, description, price, stock, created_at, updated_at FROM products ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}

	data, _ := json.Marshal(products)

	// Store in cache
	rdb.Set(ctx, cacheKey, data, cacheTTL)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

func getProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p Product
	err := db.QueryRow("SELECT id, name, description, price, stock, created_at, updated_at FROM products WHERE id = $1", id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func createProductHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var p Product
	err := db.QueryRow(
		"INSERT INTO products (name, description, price, stock) VALUES ($1, $2, $3, $4) RETURNING id, name, description, price, stock, created_at, updated_at",
		req.Name, req.Description, req.Price, req.Stock,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate cache when data changes
	rdb.Del(ctx, cacheKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func updateProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var p Product
	err := db.QueryRow(
		"UPDATE products SET name=$1, description=$2, price=$3, stock=$4, updated_at=NOW() WHERE id=$5 RETURNING id, name, description, price, stock, created_at, updated_at",
		req.Name, req.Description, req.Price, req.Stock, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rdb.Del(ctx, cacheKey)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := db.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	rdb.Del(ctx, cacheKey)

	w.WriteHeader(http.StatusNoContent)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
