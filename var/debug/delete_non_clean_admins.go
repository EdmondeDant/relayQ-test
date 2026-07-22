package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/lib/pq"
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

	for rows.Next() {
		var dbname string
		if err := rows.Scan(&dbname); err != nil {
			log.Fatal(err)
		}
		if strings.HasPrefix(dbname, "sub2api_relayq_clean_") {
			continue
		}
		dsn := fmt.Sprintf("host=localhost port=5432 user=postgres dbname=%s sslmode=disable", dbname)
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			continue
		}
		result, err := db.ExecContext(
			context.Background(),
			"delete from users where email = $1 and role = 'admin'",
			"363164954@qq.com",
		)
		_ = db.Close()
		if err != nil {
			continue
		}
		if affected, _ := result.RowsAffected(); affected > 0 {
			fmt.Printf("deleted admin in db=%s rows=%d\n", dbname, affected)
		}
	}
}
