package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Book represents a book entity
type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

var books []Book

// var channel *amqp.Channel
// var rabbitMQURL = os.Getenv("RABBITMQ_URL")
// var exchange = os.Getenv("RABBITMQ_EXCHANGE")
// var maxRetries = 10
// var retryDelay = time.Second

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

func main() {
	// API - Create a new router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/books", GetBooksHandler).Methods("GET")
	router.HandleFunc("/books/{id}", GetBookDetailsHandler).Methods("GET")

	// Start the server
	http.Handle("/", router)
	http.ListenAndServe(":8082", nil)
}
