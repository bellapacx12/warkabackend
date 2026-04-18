package storage

import (
	"encoding/json"
	"log"
	"os"

	"bingo-backend/models"
)

var Cards []models.BingoCard

func LoadCards() {
	file, err := os.ReadFile("storage/cards.json")
	if err != nil {
		log.Fatal("❌ Failed to load cards:", err)
	}

	err = json.Unmarshal(file, &Cards)
	if err != nil {
		log.Fatal("❌ Failed to parse cards:", err)
	}

	log.Println("✅ Loaded cards:", len(Cards))
}