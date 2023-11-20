package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

// Book represents a book entity
type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

var books []Book

var channel *amqp.Channel
var rabbitMQURL = os.Getenv("RABBITMQ_URL")
var exchange = os.Getenv("RABBITMQ_EXCHANGE")
var maxRetries = 10
var retryDelay = time.Second

func init() {
	// Populate some dummy data
	books = append(books, Book{ID: "1", Title: "Book 1", Author: "Author 1"})
	books = append(books, Book{ID: "2", Title: "Book 2", Author: "Author 2"})

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", "postgres", 5432, "admin", "admin", "books")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Open a connection to the database
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// Insert a new record into the user table
	newBook := Book{Title: "Book 1", Author: "Author 1"}
	//insertQuery := "INSERT INTO books(title, author) VALUES($1, $2) RETURNING id"

	// Execute the SQL query and get the ID of the inserted record
	// err = db.QueryRow(insertQuery, newBook.Title, newBook.Author).Scan(&newBook.ID)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	fmt.Printf("Inserted record with ID %s\n", newBook.ID)
}

// GetBooksHandler returns a list of books
func GetBooksHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Books handler")
	w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(books)

	// Establish a database connection
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", "postgres", 5432, "admin", "admin", "books")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Open a connection to the database
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	books, err := getBooks(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(books)
}

func getBooks(db *sql.DB) ([]Book, error) {
	rows, err := db.Query(`SELECT * FROM "books"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var book Book
		err := rows.Scan(&book.ID, &book.Author, &book.Title)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, nil
}

func getBook(db *sql.DB, id string) (Book, error) {
	var book Book
	rows, err := db.Query(fmt.Sprintf(`SELECT * FROM "books" where "id" = '%s'`, id))
	if err != nil {
		return book, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&book.ID, &book.Author, &book.Title)
		if err != nil {
			return book, err
		}
	}

	return book, nil
}

// GetBookDetailsHandler returns details about a specific book
func GetBookDetailsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the book ID from the request parameters
	params := mux.Vars(r)
	bookID := params["id"]

	// Establish a database connection
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", "postgres", 5432, "admin", "admin", "books")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Open a connection to the database
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	book, err := getBook(db, bookID)

	// If the book is not found, return a 404 Not Found response
	if err != nil {
		http.NotFound(w, r)
		return
	}

	json.NewEncoder(w).Encode(book)
}

func publishMessage(ch *amqp.Channel, exchange, routingKey, body string) {
	err := ch.Publish(
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	if err != nil {
		log.Fatalf("Failed to publish a message: %v", err)
	}
	fmt.Printf(" [x] Sent: %s\n", body)
}

func PostBookLikesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the book ID from the request parameters
	params := mux.Vars(r)
	bookID := params["id"]

	publishMessage(channel, exchange, "", bookID)

	successMessage := map[string]string{
		"success": bookID,
	}

	json.NewEncoder(w).Encode(successMessage)
}

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
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Println("Error connecting to RabbitMQ:", err)
		return false
	}
	//defer conn.Close()

	// ch, err := conn.Channel()
	// if err != nil {
	// 	log.Println("Error creating channel:", err)
	// 	return false
	// }
	// defer ch.Close()

	// You can add more specific checks based on your requirements.
	// For example, checking if a specific queue or exchange exists.

	channel, err = conn.Channel()
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

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	return true
}

func main() {
	// API - Create a new router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/books", GetBooksHandler).Methods("GET")
	router.HandleFunc("/books/{id}", GetBookDetailsHandler).Methods("GET")
	router.HandleFunc("/books/{id}/likes", PostBookLikesHandler).Methods("POST")
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// Start the server
	http.Handle("/", router)
	http.ListenAndServe(":8082", nil)
}
