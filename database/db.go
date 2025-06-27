package database

import (
	"context"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"

	"github.com/icodeologist/disasterwatch/models"
	"github.com/redis/go-redis/v9"
)

var DB *gorm.DB
var err error

func Connect() {
	host := os.Getenv("HOST")
	dbPort := os.Getenv("DBPORT")
	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbName := os.Getenv("NAME")

	//connecting to database
	dbUri := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%v port=%v", host, user, dbName, password, dbPort)

	//open the connection to database
	DB, err = gorm.Open(postgres.Open(dbUri), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Successfully connected to database")
	}

	DB.AutoMigrate(&models.Report{}, &models.User{}, &models.PasswordResetToken{}, &models.NotificationEvent{})

}

func ConnectToRedis() (*redis.Client, context.Context, error) {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ping, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, ctx, err
	}
	log.Printf("Ping : %v\n", ping)

	return client, ctx, nil

}
