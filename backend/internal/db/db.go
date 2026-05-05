package db

import (
	"fmt"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
	"os"
)

var DB *gorm.DB

func Connect() error {
	// os.Getenv, err := godotenv.Read(".env")
	// if err != nil {
	// 	return err
	// }
	host := os.Getenv("HOST")
	dbPort := os.Getenv("DBPORT")
	user := os.Getenv("DBUSER")
	password := os.Getenv("PASSWORD")
	dbName := os.Getenv("NAME")

	//connecting to database
	dbUri := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%v port=%v", host, user, dbName, password, dbPort)

	//open the connection to database
	var err error
	DB, err = gorm.Open(postgres.Open(dbUri), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to DB", "error", err)
		return err
	} else {
		slog.Info("Successfully connected to databse")
	}

	DB.AutoMigrate(&models.Report{}, &models.User{}, &models.Jobs{}, &models.DLQJob{})
	return nil

}

// Basic boilerplate to connect go with redis
func ConnectToRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		Protocol: 2,
	})
	return client
}
