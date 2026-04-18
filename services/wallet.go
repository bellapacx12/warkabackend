package services

import (
	"bingo-backend/config"
	"log"
)

func GetBalance(userID int) float64 {
	var balance float64

	err := config.DB.QueryRow(
		"SELECT balance FROM users WHERE id=$1",
		userID,
	).Scan(&balance)

	if err != nil {
		log.Println(err)
		return 0
	}

	return balance
}
func DeductBalance(userID int, amount float64) bool {
	var balance float64

	err := config.DB.QueryRow(
		"SELECT balance FROM users WHERE id=$1",
		userID,
	).Scan(&balance)

	if err != nil || balance < amount {
		return false
	}

	_, err = config.DB.Exec(
		"UPDATE users SET balance = balance - $1 WHERE id=$2",
		amount, userID,
	)

	return err == nil
}
func AddBalance(userID int, amount float64) {
	_, err := config.DB.Exec(
		"UPDATE users SET balance = balance + $1 WHERE id=$2",
		amount, userID,
	)

	if err != nil {
		log.Println(err)
	}
}