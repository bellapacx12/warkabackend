package game

import (
	"encoding/json"
	"log"
	"sync"
)

type RoomManager struct {
	Rooms map[float64]*Room
	Mutex sync.Mutex
}

var Manager = &RoomManager{
	Rooms: make(map[float64]*Room),
}
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

	copy := make(map[float64]*Room)
	for k, v := range m.Rooms {
		copy[k] = v
	}

	return copy
}
func (m *RoomManager) LobbySnapshot() []map[string]interface{} {

	m.Mutex.Lock()
	roomsCopy := make(map[float64]*Room)
	for k, v := range m.Rooms {
		roomsCopy[k] = v
	}
	m.Mutex.Unlock()

	defaultStakes := []float64{10, 20, 50, 100}

	result := []map[string]interface{}{}

	for _, stake := range defaultStakes {
		room, ok := roomsCopy[stake]
		if !ok || room == nil {
			continue
		}

		room.Mutex.Lock()

		players := len(room.Players)
		state := room.State
		countdown := room.Countdown

		room.Mutex.Unlock()

		result = append(result, map[string]interface{}{
			"stake":     stake,
			"players":   players,
			"win":       float64(players) * stake * 0.8,
			"status":    state,
			"countdown": countdown,
			"jackpot":   stake * 50,
		})
	}

	return result
}
func (m *RoomManager) FindPlayerRoom(userID int) *Room {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for _, room := range m.Rooms {
		room.Mutex.Lock()
		_, exists := room.Players[userID]
		room.Mutex.Unlock()

		if exists {
			return room
		}
	}

	return nil
}
func BroadcastLobby() {

	rooms := Manager.LobbySnapshot()

	msg := map[string]interface{}{
		"type": "rooms",
		"data": rooms,
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Println("❌ Lobby marshal error:", err)
		return
	}

	// 🔥 broadcast via game connections only
	for _, room := range Manager.Rooms {

		room.Mutex.Lock()

		for _, player := range room.Players {
			if player.Connected {
				select {
				case player.Send <- bytes:
				default:
					// avoid blocking slow clients
				}
			}
		}

		room.Mutex.Unlock()
	}
}