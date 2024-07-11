package main

import (
	"log"

	"Tonton.in-Api/api/db"
	"Tonton.in-Api/api/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	e := echo.New()

	db.InitDB()

	routes.SetupRoutes(e, db.DB)

	config := middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}

	e.Use(middleware.CORSWithConfig(config))

	e.Start(":8080")
}
