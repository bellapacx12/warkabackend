package websocket

import (
	"bingo-backend/game"
	"log"
	"net/http"
)

func HandleLobbyWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Lobby WS error:", err)
		return
	}

	game.RegisterLobby(conn)

	// 🔥 SEND INITIAL DATA IMMEDIATELY
	go game.BroadcastLobby()

	defer func() {
		game.UnregisterLobby(conn)
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Lobby WS disconnected")
			break
		}
	}
}