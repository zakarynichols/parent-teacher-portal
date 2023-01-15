package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	_ "github.com/lib/pq"
)

func main() {
	mux := mux.NewRouter()
	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux.Handle("/", http.HandlerFunc(handleRoot))
	mux.Handle("/now", http.HandlerFunc(handleNow(db)))
	mux.Handle("/user/{username}", http.HandlerFunc(handleUser(db)))
	mux.Handle("/user", http.HandlerFunc(queryUser(db, 1)))

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders: []string{"*"},
		MaxAge:         86400,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: c.Handler(mux),
	}

	log.Println("Starting server on port :" + port)
	log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
}

func openDB() (*sql.DB, error) {
	return sql.Open("postgres", "user=user password=password dbname=dbname host=db sslmode=disable")
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	responseData := map[string]string{"message": "Hello, World!"}
	jsonData, err := json.Marshal(responseData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

type currentTimeResponse struct {
	CurrentTime string `json:"current_time"`
}

func handleNow(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var currentTime time.Time
		err := db.QueryRow("SELECT NOW()").Scan(&currentTime)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		responseData := currentTimeResponse{currentTime.String()}
		jsonData, err := json.Marshal(responseData)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

type user struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func handleUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		username := vars["username"]

		var user user
		err := db.QueryRow(`SELECT id, username, password, email, role FROM users WHERE username = $1`, username).Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)

		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		jsonData, err := json.Marshal(user)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	}
}

func queryUser(db *sql.DB, id int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u user
		err := json.NewDecoder(r.Body).Decode(&u)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		row := db.QueryRow("SELECT * FROM users WHERE id=$1", u.ID)
		err = row.Scan(&u.ID, &u.Username, &u.Password, &u.Email, &u.Role)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	}
}
