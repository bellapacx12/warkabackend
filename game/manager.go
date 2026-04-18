package game

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type RoomManager struct {
	Rooms map[float64]*Room
	Mutex sync.Mutex
}

var Manager = &RoomManager{
	Rooms: make(map[float64]*Room),
}

// ==========================
// LOBBY CLIENTS
// ==========================
var LobbyClients = make(map[*websocket.Conn]bool)
var LobbyMutex sync.Mutex

func RegisterLobby(conn *websocket.Conn) {
	LobbyMutex.Lock()
	defer LobbyMutex.Unlock()

	LobbyClients[conn] = true
	log.Println("📡 Lobby client connected:", len(LobbyClients))
}

func UnregisterLobby(conn *websocket.Conn) {
	LobbyMutex.Lock()
	defer LobbyMutex.Unlock()

	delete(LobbyClients, conn)
	log.Println("❌ Lobby client disconnected:", len(LobbyClients))
}

// ==========================
// ROOM MANAGEMENT
// ==========================
func (m *RoomManager) GetRoom(stake float64) *Room {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if room, ok := m.Rooms[stake]; ok {
		return room
	}

	room := NewRoom(stake)
	m.Rooms[stake] = room

	return room
}

func (m *RoomManager) GetAllRooms() map[float64]*Room {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	// ⚠️ return copy to avoid race conditions
	copy := make(map[float64]*Room)
	for k, v := range m.Rooms {
		copy[k] = v
	}

	return copy
}

// ==========================
// LOBBY BROADCAST
// ==========================
func BroadcastLobby() {

	// ✅ ALWAYS ensure default rooms exist
	defaultStakes := []float64{10, 20, 50, 100}
	for _, stake := range defaultStakes {
		Manager.GetRoom(stake)
	}

	// ✅ get rooms BEFORE locking lobby
	roomsMap := Manager.GetAllRooms()

	rooms := []map[string]interface{}{} // ✅ never nil

	for stake, room := range roomsMap {

		room.Mutex.Lock()
		players := len(room.Players)
		status := room.State
		countdown := room.Countdown
		room.Mutex.Unlock()

		rooms = append(rooms, map[string]interface{}{
			"stake":     stake,
			"players":   players,
			"win":       float64(players) * stake * 0.8,
			"status":    status,
			"countdown": countdown,
			"jackpot":   stake * 50,
		})
	}

	msg := map[string]interface{}{
		"type": "rooms",
		"data": rooms,
	}

	bytes, _ := json.Marshal(msg)

	// ✅ lock only for writing
	LobbyMutex.Lock()
	defer LobbyMutex.Unlock()

	for conn := range LobbyClients {
		err := conn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			log.Println("❌ Lobby write error:", err)

			conn.Close()
			delete(LobbyClients, conn) // ✅ cleanup dead client
		}
	}

	log.Println("📡 Lobby broadcast:", len(rooms), "rooms →", len(LobbyClients), "clients")
}