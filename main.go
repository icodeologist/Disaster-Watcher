package main

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/routes"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	//making a connection
	database.Connect()
	db = database.Db
	//create a engine and call SetUpRoutes
	router := gin.Default()

	routes.SetUpRoutes(router)
	router.Run(":8080")
}
