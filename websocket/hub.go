package websocket

import (
	"bingo-backend/game"
	"encoding/json"
	"log"
	"net/http"

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
	UserID int     `json:"user_id"`
	CardID int     `json:"card_id"`
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		Conn: conn,
		Send: make(chan []byte),
	}

	go client.read()
	go client.write()
}

// ==========================
// READ LOOP
// ==========================
func (c *Client) read() {
	defer c.Conn.Close()

	var currentRoom *game.Room
	var userID int

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)

			if currentRoom != nil {
				currentRoom.MarkDisconnected(userID) // ✅ FIX (not RemovePlayer)
			}

			break
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("Invalid message:", err)
			continue
		}

		switch msg.Type {

		// ==========================
		// JOIN ROOM
		// ==========================
		case "join":
			userID = msg.UserID

			if msg.Stake <= 0 {
				log.Println("❌ Invalid stake:", msg.Stake)
				continue
			}

			room := game.Manager.GetRoom(msg.Stake)
			currentRoom = room

			player := &game.Player{
				UserID: msg.UserID,
				Conn:   c.Conn,
			}

			room.AddPlayer(player)

			log.Println("✅ User joined stake:", msg.Stake)

		// ==========================
		// SELECT CARD
		// ==========================
		case "select_card":
			if currentRoom == nil {
				continue
			}

			currentRoom.HandleSelectCard(msg.UserID, msg.CardID)

		default:
			log.Println("⚠️ Unknown message type:", msg.Type)
		}
	}
}

// ==========================
// WRITE LOOP (OPTIONAL)
// ==========================
func (c *Client) write() {
	defer c.Conn.Close()

	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}