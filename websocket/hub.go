package websocket

import (
	"bingo-backend/game"
	"bingo-backend/storage"
	"bingo-backend/utils"
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
    // ✅ FETCH USER FROM DB
user, err := storage.GetUserByID(int64(userID))
if err != nil {
	log.Println("❌ Failed to load user:", err)
	http.Error(w, "User not found", http.StatusUnauthorized)
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
		Username: user.Name, // ✅ THIS is your DB name
		Conn:   conn,
		Send:   sendChan,
	}
    player.SendJSON("balance", user.Balance)
	// 🔥 START WRITE PUMP (ONLY WRITER)
	go player.WritePump()
    

// ==========================
// 🔥 RESTORE SESSION (ADD THIS HERE)
// ==========================
// ==========================
// 🔥 RESTORE SESSION
// ==========================
room := game.Manager.FindPlayerRoom(player.UserID)

if room != nil {
	room.Mutex.Lock()

	_, stillExists := room.Players[player.UserID]
	state := room.State
	stake := room.Stake
    
	// copy state safely
		called := append([]int{}, room.Called...)
		countdown := room.Countdown

	room.Mutex.Unlock()

	// ✅ ONLY if real active game
	if stillExists && state != "finished" {
		log.Printf("♻️ Active game for user %d\n", player.UserID)
        
		room.ReconnectPlayer(player)
		// 🔥 tell frontend to show REJOIN
		player.SendJSON("active_game", map[string]interface{}{
			"stake": stake,
			"state": state,
		})
        // 🔥 full sync state
			player.SendJSON("init", map[string]interface{}{
				"called":    called,
				"countdown": countdown,
				"card":      room.GetPlayerCard(player.UserID),
			})
		// 🔥 reconnect player
		// OPTIONAL: replay numbers as live stream (CRITICAL FIX)
			for _, n := range called {
				player.SendJSON("number", n)
			}
         // restore card
			if card := room.GetPlayerCard(player.UserID); card != nil {
				log.Println(card)
				player.SendJSON("card", map[string]interface{}{
					"grid": card,
				})
			}

	}
}
	// 🔥 START READ LOOP
	go readLoop(conn, player)
}

// ==========================
// READ LOOP
// ==========================
func readLoop(conn *websocket.Conn, player *game.Player) {
	defer func() {
		log.Println("❌ Disconnected:", player.UserID)

		if player != nil {
			player.Connected = false
		}

		close(player.Send)
		conn.Close()
	}()

	var currentRoom *game.Room

	// 🔥 heartbeat setup
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 🔥 ping sender (VERY IMPORTANT)
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		var msg IncomingMessage

		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Read error:", err)

			if currentRoom != nil {
				currentRoom.MarkDisconnected(player.UserID)
			}
			return
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

		// ==========================
		// 🔥 BINGO (NEW)
		// ==========================
	     case "bingo":
	if currentRoom == nil {
		continue
	}

	if currentRoom.State != "playing" {
		continue
	}

	log.Println("🎉 BINGO from:", player.UserID)

	card := currentRoom.GetPlayerCard(player.UserID)

	currentRoom.Broadcast("winner", map[string]interface{}{
		"user_id":  player.UserID,
		"name": player.Username,
		"card":     card,
		"stake":    currentRoom.Stake,
	})

	currentRoom.Mutex.Lock()
	currentRoom.State = "finished"
	currentRoom.Mutex.Unlock()


// ✅ 🔥 NOTIFY FRONTEND (THIS IS WHAT YOU MISSED)
currentRoom.Broadcast("game_finished", nil)

	go func(r *game.Room) {
		time.Sleep(5 * time.Second)
		r.ResetRound()
	}(currentRoom)

		default:
			log.Println("⚠️ Unknown message:", msg.Type)
		}
	}
}