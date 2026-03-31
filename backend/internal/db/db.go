package db

import (
	"fmt"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() error {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		return err
	}
	host := envFile["HOST"]
	dbPort := envFile["DBPORT"]
	user := envFile["DBUSER"]
	password := envFile["PASSWORD"]
	dbName := envFile["NAME"]

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
