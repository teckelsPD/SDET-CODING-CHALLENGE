// consumer/main.go

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
)

var maxRetries = 10
var retryDelay = time.Second

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if err := waitForRabbitMQ(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "RabbitMQ is not reachable: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func waitForRabbitMQ() error {
	for retry := 0; retry < maxRetries; retry++ {
		log.Printf("Trying to connect to RabbitMQ (Attempt %d)...", retry+1)

		if isRabbitMQReady() {
			log.Println("RabbitMQ is ready!")
			return nil
		}

		log.Printf("RabbitMQ not ready, retrying in %v...", retryDelay)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("exhausted all retry attempts to connect to RabbitMQ")
}

func isRabbitMQReady() bool {
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	queue := os.Getenv("RABBITMQ_QUEUE")

	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	//defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	//defer channel.Close()

	err = channel.ExchangeDeclare(
		exchange, // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare an exchange: %v", err)
	}

	_, err = channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	err = channel.QueueBind(
		queue,    // queue name
		"",       // routing key
		exchange, // exchange
		false,
		nil)
	if err != nil {
		log.Fatalf("Failed to bind a queue: %v", err)
	}

	msgs, err := channel.Consume(
		queue, // queue
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	for {
		select {
		case <-signals:
			fmt.Println("Interrupt signal received, exiting...")
			return false
		case msg := <-msgs:
			fmt.Printf("Received a message: %s (Delivery Tag: %d)\n", msg.Body, msg.DeliveryTag)
		}
	}
}

func main() {
	// API - Create a new router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// Start the server
	http.Handle("/", router)
	http.ListenAndServe(":8083", nil)
}
