package routes

import (
	"database/sql"

	"Tonton.in-Api/api/controllers"

	"github.com/labstack/echo/v4"
)

func SetupRoutes(e *echo.Echo, db *sql.DB) {
	Users := controllers.NewUserController(db)
	Auth := controllers.NewAuthController(db)

	e.GET("/getallvideos", Users.GetAllVideos)
	e.GET("/getallvideosbokepindo", Users.GetAllVideosBokepIndo)
	e.GET("/getallvideosbokepliverecord", Users.GetAllVideosBokepLiveRecord)
	e.GET("/getvideos/:id", Users.GetVideosByID)
	e.POST("/videos", Users.CreateVideos)
	e.PUT("/videos/:id", Users.EditVideos)
	e.DELETE("/videos/:id", Users.DeleteVideos)
	e.PUT("/videos/inclike/:id", Users.Inclikevideos)
	e.PUT("/videos/declike/:id", Users.Declikevideos)

	e.PUT("/videos/incdislike/:id", Users.IncDislikevideos)
	e.PUT("/videos/decdislike/:id", Users.DecDislikevideos)
	e.POST("/login", Auth.Login)
	e.POST("/register", Auth.Register)
	e.DELETE("/logout", Auth.Logout)
	e.GET("/token", Auth.Token)

}
