package db

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"

	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/redis/go-redis/v9"
)

var DB *gorm.DB
var err error

func Connect() error {
	host := os.Getenv("HOST")
	dbPort := os.Getenv("DBPORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("PASSWORD")
	dbName := os.Getenv("NAME")

	//connecting to database
	dbUri := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%v port=%v", host, user, dbName, password, dbPort)

	//open the connection to database
	DB, err = gorm.Open(postgres.Open(dbUri), &gorm.Config{})
	if err != nil {
		return err
	} else {
		fmt.Println("Successfully connected to database")
	}

	DB.AutoMigrate(&models.Report{}, &models.User{})
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
