package main

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/routes"
	"net/http"
)

func main() {
	//making a connection
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	database.Connect()
	//create a engine and call SetUpRoutes
	router := gin.Default()
	routes.SetUpRoutes(router)
	router.Run(":8080")
}
