package handlers

import (
	"bingo-backend/storage"
	"bingo-backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetBalance(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")

	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing token",
		})
		return
	}

	// 🔥 extract "Bearer token"
	token := strings.TrimPrefix(authHeader, "Bearer ")

	userID, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid token",
		})
		return
	}

	user, err := storage.GetUserByID(int64(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance": user.Balance,
	})
}