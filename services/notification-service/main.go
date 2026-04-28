package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderEvent struct {
	Event     string  `json:"event"`
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	Total     float64 `json:"total"`
	Timestamp string  `json:"timestamp"`
}

func main() {
	// Health check endpoint runs in background
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})
		port := getEnv("PORT", "8084")
		log.Printf("Health endpoint on :%s", port)
		http.ListenAndServe(":"+port, mux)
	}()

	// Connect to RabbitMQ and consume events
	consumeEvents()
}

func consumeEvents() {
	rabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		getEnv("RABBITMQ_USER", "shopstream"),
		getEnv("RABBITMQ_PASSWORD", "shopstream123"),
		getEnv("RABBITMQ_HOST", "rabbitmq.shopstream-data.svc.cluster.local"),
		getEnv("RABBITMQ_PORT", "5672"),
	)

	var conn *amqp.Connection
	var err error

	// Retry connection (RabbitMQ might not be ready yet)
	for i := 0; i < 30; i++ {
		conn, err = amqp.Dial(rabbitURL)
		if err == nil {
			break
		}
		log.Printf("Waiting for RabbitMQ... attempt %d/30: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue (creates it if it doesn't exist)
	q, err := ch.QueueDeclare(
		"order.placed", // queue name
		true,           // durable (survives RabbitMQ restart)
		false,          // auto-delete
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer tag
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Println("Waiting for order events...")

	for msg := range msgs {
		var event OrderEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("Failed to parse event: %v", err)
			continue
		}

		log.Printf("📧 NOTIFICATION: Order %s placed by user %s — Total: $%.2f",
			event.OrderID, event.UserID, event.Total)
		log.Printf("   → Sending confirmation email (simulated)")
		log.Printf("   → Sending SMS notification (simulated)")
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
