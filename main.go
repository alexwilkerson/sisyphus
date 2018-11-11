package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	dbUser        = os.Getenv("SISYPHUSDBUSER")
	dbPassword    = os.Getenv("SISYPHUSDBPW")
	db            *sql.DB
	checkEmail    = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$").MatchString
	checkUsername = regexp.MustCompile("^[A-Za-z0-9_]+$").MatchString
)

const (
	dbHost = "localhost"
	dbPort = 5432
	dbName = "sisyphus"
)

type user struct {
	ID           int        `json:"id,omitempty"`
	Active       bool       `json:"active,omitempty"`
	CreationDate *time.Time `json:"creation_date,omitempty"`
	Day          int        `json:"day,omitempty"`
	Username     string     `json:"username,omitempty"`
	Password     string     `json:"password,omitempty"`
	Email        string     `json:"email,omitempty"`
	LastPush     *time.Time `json:"last_push,omitempty"`
	Fulfilled    bool       `json:"fulfilled,omitempty"`
	Secret       string     `json:"secret,omitempty"`
	Contact1     string     `json:"contact1,omitempty"`
	Contact2     string     `json:"contact2,omitempty"`
	Contact3     string     `json:"contact3,omitempty"`
	Contact4     string     `json:"contact4,omitempty"`
	Contact5     string     `json:"contact5,omitempty"`
}

type jsonError struct {
	Error string `json:"error,omitempty"`
}

func main() {
	initDB()
	defer db.Close()

	router := mux.NewRouter()
	router.HandleFunc("/api/users/create", createUserHandler).Methods("POST")
	router.HandleFunc("/api/users/{id}", getUserHandler).Methods("GET")
	router.HandleFunc("/api/users/push", pushHandler).Methods("POST")
	router.HandleFunc("/api/users/login", loginHandler).Methods("POST")
	fmt.Println("Running on port: 5042")
	log.Fatal(http.ListenAndServe(":5042", router))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var u user
	err := decoder.Decode(&u)
	if err != nil {
		writeJSONError(w, "error decoding json parameters")
		return
	}
	if u.Username == "" || u.Password == "" {
		writeJSONError(w, "empty or missing fields")
		return
	}
	if err != nil {
		writeJSONError(w, "error encrypting password")
		return
	}
	sqlStatement := `
		SELECT id, password, creation_date, active, last_push, fulfilled
		FROM users
		WHERE username = $1
	`
	var passwordFromDB []byte
	err = db.QueryRow(sqlStatement, u.Username).Scan(&u.ID, &passwordFromDB,
		&u.CreationDate, &u.Active, &u.LastPush, &u.Fulfilled)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	if err := bcrypt.CompareHashAndPassword(passwordFromDB, []byte(u.Password)); err != nil {
		writeJSONError(w, err.Error())
		return
	}

	calculateDay(&u)

	u.Password = ""
	u.CreationDate = nil
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var u user
	err := decoder.Decode(&u)
	if err != nil {
		writeJSONError(w, "error decoding json parameters")
		return
	}
	if u.Username == "" || u.Password == "" || u.Email == "" ||
		u.Secret == "" || u.Contact1 == "" || u.Contact2 == "" ||
		u.Contact3 == "" || u.Contact4 == "" || u.Contact5 == "" {
		writeJSONError(w, "empty of missing fields")
		return
	}
	switch {
	case !checkUsername(u.Username):
		writeJSONError(w, "username is invalid")
		return
	case len(u.Username) < 3 || len(u.Username) > 20:
		writeJSONError(w, "username must be between 3 and 20 characters long")
		return
	case !checkEmail(u.Email):
		writeJSONError(w, "user email is invalid")
		return
	case !checkEmail(u.Contact1):
		writeJSONError(w, "contact1 email is invalid")
		return
	case !checkEmail(u.Contact2):
		writeJSONError(w, "contact2 email is invalid")
		return
	case !checkEmail(u.Contact3):
		writeJSONError(w, "contact3 email is invalid")
		return
	case !checkEmail(u.Contact4):
		writeJSONError(w, "contact4 email is invalid")
		return
	case !checkEmail(u.Contact5):
		writeJSONError(w, "contact5 email is invalid")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, "error encrypting password")
		return
	}
	sqlStatement := `
		INSERT INTO users (creation_date, username, password, email,
		last_push, secret, contact1, contact2, contact3, contact4, contact5)
		VALUES (NOW(), $1, $2, $3, NOW(), $4, $5, $6, $7, $8, $9)
		RETURNING id, creation_date, active, username, email, last_push, fulfilled,
		secret, contact1, contact2, contact3, contact4, contact5
	`
	err = db.QueryRow(sqlStatement, u.Username, hash, u.Email,
		u.Secret, u.Contact1, u.Contact2, u.Contact3, u.Contact4,
		u.Contact5).Scan(&u.ID, &u.CreationDate, &u.Active, &u.Username,
		&u.Email, &u.Fulfilled, &u.LastPush, &u.Secret, &u.Contact1, &u.Contact2,
		&u.Contact3, &u.Contact4, &u.Contact5)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	u.Password = ""
	calculateDay(&u)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	sqlStatement := `
		SELECT id, creation_date, active, username, email, last_push, fulfilled, secret, contact1, contact2, contact3, contact4, contact5
		FROM users
		wHERE id = $1
	`
	var u user
	err := db.QueryRow(sqlStatement, id).Scan(&u.ID,
		&u.CreationDate, &u.Active, &u.Username, &u.Email,
		&u.LastPush, &u.Fulfilled, &u.Secret, &u.Contact1, &u.Contact2,
		&u.Contact3, &u.Contact4, &u.Contact5)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	calculateDay(&u)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var u user
	err := decoder.Decode(&u)
	if err != nil {
		writeJSONError(w, "error decoding json parameters")
		return
	}
	if u.ID == 0 || u.Password == "" {
		writeJSONError(w, "empty of missing fields")
		return
	}
	if err != nil {
		writeJSONError(w, "error encrypting password")
		return
	}
	sqlStatement := `
		SELECT id, creation_date, active, username, password, email,
		last_push, fulfilled, secret, contact1, contact2, contact3, contact4, contact5
		FROM users
		WHERE id = $1
	`
	var passwordFromDB []byte
	err = db.QueryRow(sqlStatement, u.ID).Scan(&u.ID, &u.CreationDate,
		&u.Active, &u.Username, &passwordFromDB, &u.Email, &u.LastPush,
		&u.Fulfilled, &u.Secret, &u.Contact1, &u.Contact2,
		&u.Contact3, &u.Contact4, &u.Contact5)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	if err := bcrypt.CompareHashAndPassword(passwordFromDB, []byte(u.Password)); err != nil {
		writeJSONError(w, err.Error())
		return
	}

	sqlStatement = `
		UPDATE users
		SET last_push = NOW(), fulfilled = true
		WHERE id = $1
		RETURNING last_push, fulfilled
	`
	err = db.QueryRow(sqlStatement, u.ID).Scan(&u.LastPush, &u.Fulfilled)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}

	u.Password = ""
	calculateDay(&u)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func writeJSONError(w http.ResponseWriter, e string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(jsonError{Error: e})
}

func calculateDay(u *user) {
	date := time.Now().Local()
	diff := date.Sub(*u.CreationDate)
	u.Day = int(diff.Hours()/24) + 1
}

func initDB() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connection to database successful.")
}
