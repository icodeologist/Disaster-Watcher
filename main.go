package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/routes"
)

// everything about main here
// set up and calling all the essentials

func main() {
	db.Connect()
	fmt.Println("db connected")
	router := gin.Default()
	routes.SetUpRoutes(router)
	router.Run(":3000")

}
