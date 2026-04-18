package handlers

import (
	"bingo-backend/models"
	"bingo-backend/storage"
	"bingo-backend/utils"
	"database/sql"
	"errors"
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

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid"})
		return
	}

	user, err := storage.GetUserByTelegramID(body.TelegramID)

	// ✅ ONLY create if user truly does not exist
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {

			newUser := &models.User{
				TelegramID: body.TelegramID,
				Name:       body.Name,
				Phone:      body.Phone,
				Balance:    0,
			}

			user, err = storage.CreateUser(newUser)
			if err != nil {
				log.Println("CREATE USER ERROR:", err)

c.JSON(500, gin.H{
	"error": err.Error(), // 🔥 show real error
})
				return
			}

		} else {
			// ❌ real DB error
			c.JSON(500, gin.H{"error": "database error"})
			return
		}
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

