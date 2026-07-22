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

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

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
		result, err := db.ExecContext(
			context.Background(),
			"update users set password_hash = $1 where email = $2",
			string(hash),
			"363164954@qq.com",
		)
		_ = db.Close()
		if err != nil {
			continue
		}
		if affected, _ := result.RowsAffected(); affected > 0 {
			fmt.Printf("reset password in db=%s rows=%d\n", dbname, affected)
		}
	}
}
