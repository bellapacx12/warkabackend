package websocket

import (
	"encoding/json"
	"log"

	"bingo-backend/game"
	"bingo-backend/storage"
)

func sendAvailableCards(c *Client, room *game.Room) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	available := []interface{}{}

	for _, card := range storage.Cards {
		if !room.UsedCards[card.CardID] {
			available = append(available, card)
		}
	}

	msg := map[string]interface{}{
		"type": "cards",
		"data": available,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Println("❌ marshal error:", err)
		return
	}

	c.Conn.WriteMessage(1, bytes)
}