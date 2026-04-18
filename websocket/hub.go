package websocket

import (
	"bingo-backend/game"
	"bingo-backend/utils"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

type IncomingMessage struct {
	Type   string  `json:"type"`
	Stake  float64 `json:"stake"`
	CardID int     `json:"card_id"`
}

// ==========================
// ENTRY
// ==========================
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	userID, err := utils.ValidateToken(token)
	if err != nil {
		log.Println("❌ Invalid token:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	sendChan := make(chan []byte, 256)

	// 🔥 CREATE PLAYER (IMPORTANT)
	player := &game.Player{
		UserID: int(userID),
		Conn:   conn,
		Send:   sendChan,
	}

	// 🔥 START WRITE PUMP (ONLY WRITER)
	go player.WritePump()

	// 🔥 START READ LOOP
	go readLoop(conn, player)
}

// ==========================
// READ LOOP
// ==========================
func readLoop(conn *websocket.Conn, player *game.Player) {
	defer func() {
		close(player.Send)
		conn.Close()
	}()

	var currentRoom *game.Room

	// 🔥 heartbeat
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)

			if currentRoom != nil {
				currentRoom.MarkDisconnected(player.UserID)
			}
			return
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {

		// ==========================
		// JOIN ROOM
		// ==========================
		case "join":
			if msg.Stake <= 0 {
				continue
			}

			room := game.Manager.GetRoom(msg.Stake)
			currentRoom = room

			room.AddPlayer(player)

			log.Printf("✅ User %d joined stake %.0f\n", player.UserID, msg.Stake)

		// ==========================
		// SELECT CARD
		// ==========================
		case "select_card":
			if currentRoom == nil {
				continue
			}

			currentRoom.HandleSelectCard(player.UserID, msg.CardID)

		default:
			log.Println("⚠️ Unknown message:", msg.Type)
		}
	}
}