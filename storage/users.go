package storage

import (
	"bingo-backend/config"
	"bingo-backend/models"
)

func GetUserByTelegramID(tgID int64) (*models.User, error) {
	row := config.DB.QueryRow(`
		SELECT id, telegram_id, name, phone, balance
		FROM users
		WHERE telegram_id = $1
	`, tgID)

	var user models.User
	err := row.Scan(
		&user.ID,
		&user.TelegramID,
		&user.Name,
		&user.Phone,
		&user.Balance,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func CreateUser(u *models.User) (*models.User, error) {
	err := config.DB.QueryRow(`
		INSERT INTO users (telegram_id, name, phone, balance)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		u.TelegramID,
		u.Name,
		u.Phone,
		u.Balance,
	).Scan(&u.ID)

	if err != nil {
		return nil, err
	}

	return u, nil
}
func GetUserByID(id int64) (*models.User, error) {
	row := config.DB.QueryRow(`
		SELECT id, telegram_id, name, phone, balance
		FROM users
		WHERE id = $1
	`, id)

	var user models.User
	err := row.Scan(
		&user.ID,
		&user.TelegramID,
		&user.Name,
		&user.Phone,
		&user.Balance,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}