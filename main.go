package main

import (
	"Tonton.in-Api/api/db"
	"Tonton.in-Api/api/routes"

	"github.com/labstack/echo/v4"
	_ "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	db.InitDB()

	routes.SetupRoutes(e, db.DB)

	config := middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}

	e.Use(middleware.CORSWithConfig(config))

	e.Start(":8080")
}
