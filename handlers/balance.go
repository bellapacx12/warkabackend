package handlers

import (
	"bingo-backend/storage"
	"bingo-backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetBalance(c *gin.Context) {
	token := c.Query("token")

	userID, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := storage.GetUserByID(int64(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance": user.Balance,
	})
}