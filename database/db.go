package database

import (
	"fmt"
	"log"
	"os"

	"github.com/icodeologist/disasterwatch/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	DB.AutoMigrate(&models.Report{}, &models.User{}, &models.PasswordResetToken{})

}
