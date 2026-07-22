package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
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
	user := viper.GetString("database.user")
	if user == "" {
		user = "postgres"
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
	serverPort := viper.GetInt("server.port")
	if serverPort == 0 {
		serverPort = 8080
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, dbname, sslmode)
	fmt.Printf("config: db=%s host=%s port=%d server_port=%d\n", dbname, host, port, serverPort)

	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	users, err := client.User.Query().Order(ent.Asc("id")).Limit(20).All(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		passwordOK := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("admin123456")) == nil
		fmt.Printf("user: id=%d email=%s role=%s status=%s signup_source=%s has_password=%t admin123456_ok=%t\n", u.ID, u.Email, u.Role, u.Status, u.SignupSource, u.PasswordHash != "", passwordOK)
	}
}
