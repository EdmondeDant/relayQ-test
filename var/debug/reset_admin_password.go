package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/ent/user"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	host := viper.GetString("database.host")
	if host == "" {
		host = "localhost"
	}
	port := viper.GetInt("database.port")
	if port == 0 {
		port = 5432
	}
	dbUser := viper.GetString("database.user")
	if dbUser == "" {
		dbUser = "postgres"
	}
	password := viper.GetString("database.password")
	dbname := viper.GetString("database.dbname")
	if dbname == "" {
		dbname = "sub2api"
	}
	sslmode := viper.GetString("database.sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", host, port, dbUser, password, dbname, sslmode)
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()
	admin, err := client.User.Query().Where(user.RoleEQ("admin"), user.StatusEQ("active")).First(ctx)
	if err != nil {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	if err := client.User.UpdateOneID(admin.ID).SetPasswordHash(string(hash)).Exec(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("reset admin password: email=%s id=%d\n", admin.Email, admin.ID)
}
