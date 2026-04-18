package models

type User struct {
	ID         int64   `json:"id"`
	TelegramID int64   `json:"telegram_id"`
	Name       string  `json:"name"`
	Phone      string  `json:"phone"`
	Balance    float64 `json:"balance"`
}