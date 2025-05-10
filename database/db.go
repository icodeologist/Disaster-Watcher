package database

import (
	"fmt"
	"log"
	"os"

	"github.com/icodeologist/disasterwatch/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Db *gorm.DB
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
	Db, err = gorm.Open(postgres.Open(dbUri), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Successfully connected to database")
	}

	Db.AutoMigrate(&models.Report{})

}
