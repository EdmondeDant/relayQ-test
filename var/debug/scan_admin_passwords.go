package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	rootDSN := "host=localhost port=5432 user=postgres dbname=postgres sslmode=disable"
	rootDB, err := sql.Open("postgres", rootDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer rootDB.Close()

	rows, err := rootDB.QueryContext(context.Background(), "select datname from pg_database where datistemplate = false order by datname")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var dbname string
	for rows.Next() {
		if err := rows.Scan(&dbname); err != nil {
			log.Fatal(err)
		}
		dsn := fmt.Sprintf("host=localhost port=5432 user=postgres dbname=%s sslmode=disable", dbname)
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			continue
		}
		var email, role, status, signupSource, passwordHash string
		err = db.QueryRowContext(
			context.Background(),
			"select email, role, status, coalesce(signup_source, ''), password_hash from users where email = $1 limit 1",
			"363164954@qq.com",
		).Scan(&email, &role, &status, &signupSource, &passwordHash)
		_ = db.Close()
		if err != nil {
			continue
		}
		passwordOK := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("admin123456")) == nil
		fmt.Printf("db=%s email=%s role=%s status=%s signup_source=%s admin123456_ok=%t\n", dbname, email, role, status, signupSource, passwordOK)
	}
}
