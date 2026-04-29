package main

import (
	"log"
	"os"
	"time"

	"bingo-backend/config"
	"bingo-backend/handlers"
	"bingo-backend/middleware"
	"bingo-backend/storage"
	"bingo-backend/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	storage.LoadCards()
	config.ConnectDB()

	r := gin.Default()

	// 🌐 CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://warkafrontend.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 🟢 Public routes
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	r.POST("/auth/telegram", handlers.RegisterTelegramUser)
    r.GET("/auth/telegram", handlers.GetTelegramUser)
	// 🔐 Protected routes
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())

	auth.GET("/rooms", handlers.GetRooms)

	auth.GET("/me", func(c *gin.Context) {
	userID := c.GetInt64("user_id")
   
	user, err := storage.GetUserByID(userID)
	if err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	c.JSON(200, user)
})

	// 🔌 WebSockets (token via query param)
	r.GET("/ws", func(c *gin.Context) {
		websocket.HandleWebSocket(c.Writer, c.Request)
	})

	

	log.Println("Server running on :8080")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}