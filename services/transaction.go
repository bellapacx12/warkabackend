package services

import "bingo-backend/config"

func CreateTransaction(userID int, amount float64, ttype string) {
	config.DB.Exec(
		"INSERT INTO transactions (user_id, amount, type) VALUES ($1,$2,$3)",
		userID, amount, ttype,
	)
}