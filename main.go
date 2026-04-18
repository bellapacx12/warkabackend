package main

import (
	"log"
	"time"

	"bingo-backend/config"
	"bingo-backend/handlers"
	"bingo-backend/storage"
	"bingo-backend/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	storage.LoadCards()
    config.ConnectDB()
	r := gin.Default()
    
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // Next.js dev
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge: 12 * time.Hour,
	}))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	r.GET("/ws", func(c *gin.Context) {
		websocket.HandleWebSocket(c.Writer, c.Request)
	})
   r.GET("/rooms", handlers.GetRooms)
   r.GET("/ws/lobby", func(c *gin.Context) {
	websocket.HandleLobbyWS(c.Writer, c.Request)
})
	log.Println("Server running on :8080")
	r.Run(":8080")
}