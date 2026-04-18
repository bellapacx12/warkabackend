package models

type BingoCard struct {
	B      []*int `json:"B"`
	I      []*int `json:"I"`
	N      []*int `json:"N"`
	G      []*int `json:"G"`
	O      []*int `json:"O"`
	CardID int    `json:"card_id"`
}