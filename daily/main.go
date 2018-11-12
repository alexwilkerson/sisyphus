package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var (
	from             = os.Getenv("SISYPHUSEMAIL")
	pass             = os.Getenv("SISYPHUSEMAILPW")
	dbUser           = os.Getenv("SISYPHUSDBUSER")
	dbPassword       = os.Getenv("SISYPHUSDBPW")
	db               *sql.DB
	unfulfilledUsers []user
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

func main() {
	initDB()
	defer db.Close()

	fmt.Println(time.Now())

	sqlStatement := `
		SELECT id, active, username, email, last_push,
			secret, contact1, contact2, contact3, contact4, contact5
		FROM users
		WHERE active = true
	`
	rows, err := db.Query(sqlStatement)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var u user
		err = rows.Scan(&u.ID, &u.Active, &u.Username, &u.Email, &u.LastPush,
			&u.Secret, &u.Contact1, &u.Contact2, &u.Contact3,
			&u.Contact4, &u.Contact5)
		print("yes")
		if calculateHours(*u.LastPush) > 24 {
			unfulfilledUsers = append(unfulfilledUsers, u)
		}
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	// send("alex@uncorrected.com", "hello there")
}

func send(to, body string) {
	from := os.Getenv("SISYPHUSEMAIL")
	pass := os.Getenv("SISYPHUSEMAILPW")

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Sisyphus Greets You\n\n" +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		panic(err)
	}

	log.Print("email sent.")
}

func calculateHours(t time.Time) int {
	elapsed := time.Since(t)
	print(int(elapsed.Hours()))
	return int(elapsed.Hours())
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
