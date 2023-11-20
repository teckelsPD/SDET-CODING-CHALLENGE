package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// BackendService represents a backend service
type BackendService struct {
	Name      string
	URL       string
	Protected bool
}

// Define the secret key for JWT
var secretKey = []byte("secret")

// CustomClaims represents the claims to be encoded in the JWT
type CustomClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// User represents the claims to be encoded in the JWT
type User struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	ID       string `json:"id"`
	Password string `json:"password"`
}

var backendServices = map[string]BackendService{
	"profile": {"profile", "http://profile-service:8081", true},
	"books":   {"books", "http://books-service:8082", false},
	// Add more backend services as needed
}

// GenerateJWT generates a new JWT token
func GenerateJWT(username string) (string, error) {
	claims := &CustomClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 1).Unix(), // Token expires in 1 hour
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func reverseProxy(targetURL string) *httputil.ReverseProxy {
	target, _ := url.Parse(targetURL)
	fmt.Printf("Parsed %s\n", target)
	return httputil.NewSingleHostReverseProxy(target)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Handle request\n")
	vars := mux.Vars(r)
	serviceName := vars["serviceID"]
	fmt.Printf("ServiceName: %s\n", serviceName)
	service, exists := backendServices[serviceName]
	if !exists {
		fmt.Printf("Service not found\n")
		http.NotFound(w, r)
		return
	}

	if service.Protected {
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract the token from the Authorization header
		tokenString := strings.Replace(authorizationHeader, "Bearer ", "", 1)

		// Parse the JWT token
		token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the token is valid
		if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
			fmt.Printf("Authenticated user: %s\n", claims.Username)
			r.Header.Add("username", claims.Username)
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}

	backendURL := service.URL
	fmt.Printf("Routing request to %s (%s)\n", service.Name, backendURL)

	// Forward the request to the corresponding backend service
	reverseProxy := reverseProxy(backendURL)
	reverseProxy.ServeHTTP(w, r)
}

func getUser(username string) (User, error) {
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

	var user User

	rows, err := db.Query(fmt.Sprintf(`SELECT * FROM "users" WHERE "username" = '%s';`, username))
	if err != nil {
		return user, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&user.Username, &user.ID, &user.Password)
		if err != nil {
			return user, err
		}
	}

	return user, nil
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form data", http.StatusBadRequest)
			return
		}

		// Access form values using r.Form
		username := r.Form.Get("username")
		// password := r.Form.Get("password")

		// Perform authentication (e.g., check credentials against a database)
		// For simplicity, assume any valid username/password pair is authenticated
		_, err = getUser(username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		token, err := GenerateJWT(username)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user := User{Username: username, Token: token}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}).Methods("POST")

	router.HandleFunc("/{serviceID}", handleRequest)
	router.HandleFunc("/{serviceID}/{rest:.*}", handleRequest)

	http.Handle("/", router)
	fmt.Println("API Gateway listening on :8080")
	http.ListenAndServe(":8080", nil)
}
