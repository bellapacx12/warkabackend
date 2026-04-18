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

	client := &Client{
		Conn: conn,
		Send: make(chan []byte, 256), // buffered 🔥
	}

	go client.read(int(userID))
	go client.write()
}

// ==========================
// READ LOOP
// ==========================
func (c *Client) read(userID int) {
	defer func() {
		close(c.Send)
		c.Conn.Close()
	}()

	var currentRoom *game.Room

	// 🔥 heartbeat setup
	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)

			if currentRoom != nil {
				currentRoom.MarkDisconnected(userID)
			}

			break
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

			player := &game.Player{
				UserID: userID,
				Send:   c.Send,   // ✅ USE CHANNEL
				Conn:   c.Conn,   // optional (for closing)
			}

			room.AddPlayer(player)

			log.Printf("✅ User %d joined stake %.0f\n", userID, msg.Stake)

		// ==========================
		// SELECT CARD
		// ==========================
		case "select_card":
			if currentRoom == nil {
				continue
			}

			currentRoom.HandleSelectCard(userID, msg.CardID)

		default:
			log.Println("⚠️ Unknown message:", msg.Type)
		}
	}
}

// ==========================
// WRITE LOOP
// ==========================
func (c *Client) write() {
	ticker := time.NewTicker(30 * time.Second) // 🔥 ping
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {

		case msg, ok := <-c.Send:
			if !ok {
				return
			}

			err := c.Conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Println("Write error:", err)
				return
			}

		case <-ticker.C:
			// 🔥 keep connection alive
			err := c.Conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		}
	}
}