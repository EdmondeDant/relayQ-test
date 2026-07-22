package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
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

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, dbname, sslmode)
	fmt.Printf("config: db=%s host=%s port=%d\n", dbname, host, port)

	client, err := ent.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	keys, err := client.APIKey.Query().WithGroup().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, key := range keys {
		groupName := ""
		groupPlatform := ""
		groupID := int64(0)
		if key.Edges.Group != nil {
			groupID = key.Edges.Group.ID
			groupName = key.Edges.Group.Name
			groupPlatform = key.Edges.Group.Platform
		}
		fmt.Printf("apikey: id=%d name=%s status=%s group_id=%d group=%s platform=%s value_prefix=%s\n",
			key.ID, key.Name, key.Status, groupID, groupName, groupPlatform, prefix(key.Key))
	}

	fmt.Println("---- groups ----")
	groups, err := client.Group.Query().Order(ent.Asc("id")).All(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, group := range groups {
		fmt.Printf("group: id=%d name=%s platform=%s status=%s\n", group.ID, group.Name, group.Platform, group.Status)
	}
}

func prefix(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 10 {
		return value
	}
	return value[:10]
}
