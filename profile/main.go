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
type Profile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// GetProfileHandler returns a profile
func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Profile handler")
	w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(me)
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

	username := r.Header.Get("username")

	var profile Profile

	rows, err := db.Query(fmt.Sprintf(`SELECT * FROM "users" WHERE "username" = '%s';`, username))
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&profile.Username, &profile.ID, &profile.Password)
		if err != nil {
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
		}
	}

	json.NewEncoder(w).Encode(profile)
}

func main() {
	// Create a new router
	router := mux.NewRouter()

	// Define API routes
	router.HandleFunc("/profile/me", GetProfileHandler).Methods("GET")

	// Start the server
	http.Handle("/", router)
	http.ListenAndServe(":8081", nil)
}
