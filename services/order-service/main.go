package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	Status    string      `json:"status"`
	Total     float64     `json:"total"`
	Items     []OrderItem `json:"items,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

type OrderItem struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type CreateOrderRequest struct {
	UserID string               `json:"user_id"`
	Items  []CreateOrderItemReq `json:"items"`
}

type CreateOrderItemReq struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Product struct {
	ID    string  `json:"id"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type OrderEvent struct {
	Event     string  `json:"event"`
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	Total     float64 `json:"total"`
	Timestamp string  `json:"timestamp"`
}

var (
	db      *sql.DB
	rabbitCh *amqp.Channel
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrations()
		return
	}

	connectDB()
	defer db.Close()
	connectRabbitMQ()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /api/orders", createOrderHandler)
	mux.HandleFunc("GET /api/orders/{id}", getOrderHandler)
	mux.HandleFunc("GET /api/orders/user/{userId}", listUserOrdersHandler)

	port := getEnv("PORT", "8082")
	log.Printf("Order service starting on :%s", port)
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

func connectRabbitMQ() {
	rabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		getEnv("RABBITMQ_USER", "shopstream"),
		getEnv("RABBITMQ_PASSWORD", "shopstream123"),
		getEnv("RABBITMQ_HOST", "rabbitmq.shopstream-data.svc.cluster.local"),
		getEnv("RABBITMQ_PORT", "5672"),
	)

	var conn *amqp.Connection
	var err error
	for i := 0; i < 30; i++ {
		conn, err = amqp.Dial(rabbitURL)
		if err == nil {
			break
		}
		log.Printf("Waiting for RabbitMQ... attempt %d/30", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("RabbitMQ not available — events will not be published: %v", err)
		return
	}

	rabbitCh, err = conn.Channel()
	if err != nil {
		log.Printf("Failed to open RabbitMQ channel: %v", err)
		return
	}

	// Declare the queue
	_, err = rabbitCh.QueueDeclare("order.placed", true, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return
	}

	log.Println("Connected to RabbitMQ")
}

func publishOrderEvent(order Order) {
	if rabbitCh == nil {
		log.Println("RabbitMQ not connected — skipping event publish")
		return
	}

	event := OrderEvent{
		Event:     "order.placed",
		OrderID:   order.ID,
		UserID:    order.UserID,
		Total:     order.Total,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	body, _ := json.Marshal(event)

	err := rabbitCh.PublishWithContext(
		context.Background(),
		"",             // exchange
		"order.placed", // routing key (queue name)
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Printf("Failed to publish event: %v", err)
		return
	}

	log.Printf("Published order.placed event for order %s", order.ID)
}

func runMigrations() {
	connectDB()
	defer db.Close()

	migration := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE TABLE IF NOT EXISTS orders (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		total DECIMAL(10,2) NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS order_items (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		order_id UUID NOT NULL REFERENCES orders(id),
		product_id UUID NOT NULL,
		quantity INTEGER NOT NULL,
		price DECIMAL(10,2) NOT NULL
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

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || len(req.Items) == 0 {
		http.Error(w, "user_id and items are required", http.StatusBadRequest)
		return
	}

	productServiceURL := getEnv("PRODUCT_SERVICE_URL", "http://product-service.shopstream.svc.cluster.local")
	var total float64
	var validatedItems []struct {
		ProductID string
		Quantity  int
		Price     float64
	}

	for _, item := range req.Items {
		resp, err := http.Get(fmt.Sprintf("%s/api/products/%s", productServiceURL, item.ProductID))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to validate product %s: %v", item.ProductID, err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			http.Error(w, fmt.Sprintf("product %s not found", item.ProductID), http.StatusBadRequest)
			return
		}

		body, _ := io.ReadAll(resp.Body)
		var product Product
		json.Unmarshal(body, &product)

		if product.Stock < item.Quantity {
			http.Error(w, fmt.Sprintf("insufficient stock for product %s", item.ProductID), http.StatusBadRequest)
			return
		}

		validatedItems = append(validatedItems, struct {
			ProductID string
			Quantity  int
			Price     float64
		}{item.ProductID, item.Quantity, product.Price})
		total += product.Price * float64(item.Quantity)
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var order Order
	err = tx.QueryRow(
		"INSERT INTO orders (user_id, status, total) VALUES ($1, 'pending', $2) RETURNING id, user_id, status, total, created_at",
		req.UserID, total,
	).Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.CreatedAt)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, item := range validatedItems {
		_, err = tx.Exec(
			"INSERT INTO order_items (order_id, product_id, quantity, price) VALUES ($1, $2, $3, $4)",
			order.ID, item.ProductID, item.Quantity, item.Price,
		)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stockUpdate, _ := json.Marshal(map[string]interface{}{
			"name": "", "description": "", "price": item.Price, "stock": -item.Quantity,
		})
		putReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/products/%s", productServiceURL, item.ProductID), bytes.NewBuffer(stockUpdate))
		putReq.Header.Set("Content-Type", "application/json")
		http.DefaultClient.Do(putReq)
	}

	tx.Commit()

	// Publish event to RabbitMQ (async — doesn't block the response)
	go publishOrderEvent(order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var order Order
	err := db.QueryRow("SELECT id, user_id, status, total, created_at FROM orders WHERE id = $1", id).
		Scan(&order.ID, &order.UserID, &order.Status, &order.Total, &order.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("SELECT id, order_id, product_id, quantity, price FROM order_items WHERE order_id = $1", id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item OrderItem
			rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price)
			order.Items = append(order.Items, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func listUserOrdersHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	rows, err := db.Query("SELECT id, user_id, status, total, created_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var o Order
		rows.Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)
		orders = append(orders, o)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
