package handlers

import (
	"bingo-backend/models"
	"bingo-backend/storage"
	"bingo-backend/utils"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

type TelegramAuthRequest struct {
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
}

func RegisterTelegramUser(c *gin.Context) {
	var body TelegramAuthRequest

	// 🔍 Bind request
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Println("BIND ERROR:", err)
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	// 🔍 Validate input
	if body.TelegramID == 0 {
		c.JSON(400, gin.H{"error": "telegram_id required"})
		return
	}

	log.Println("➡️ Incoming TelegramID:", body.TelegramID)

	// 🔍 Try to get existing user
	user, err := storage.GetUserByTelegramID(body.TelegramID)

	if err != nil {

		// ✅ User not found → create
		if errors.Is(err, sql.ErrNoRows) {

			log.Println("👤 User not found, creating...")

			newUser := &models.User{
				TelegramID: body.TelegramID,
				Name:       body.Name,
				Phone:      body.Phone,
				Balance:    0,
			}

			user, err = storage.CreateUser(newUser)
			if err != nil {
				log.Println("❌ CREATE USER ERROR:", err)

				c.JSON(500, gin.H{
					"error": err.Error(),
				})
				return
			}

			log.Println("✅ User created with ID:", user.ID)

		} else {
			// ❌ REAL DB ERROR (this is what you were hiding)
			log.Println("❌ GET USER ERROR:", err)

			c.JSON(500, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	// 🔐 Generate token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		log.Println("❌ TOKEN ERROR:", err)

		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	log.Println("✅ Auth success for user:", user.ID)

	c.JSON(200, gin.H{
		"token": token,
	})
}
func GetTelegramUser(c *gin.Context) {
	tgIDStr := c.Query("telegram_id")

	var tgID int64
	_, err := fmt.Sscan(tgIDStr, &tgID)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid telegram_id"})
		return
	}

	user, err := storage.GetUserByTelegramID(tgID)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "token error"})
		return
	}

	c.JSON(200, gin.H{
		"token": token,
	})
}