package handlers

import (
	"bingo-backend/game"
	"sort"

	"github.com/gin-gonic/gin"
)

type RoomDTO struct {
	Stake     float64 `json:"stake"`
	Players   int     `json:"players"`
	Win       float64 `json:"win"`
	Status    string  `json:"status"`
	Countdown int     `json:"countdown"`
	Jackpot   float64 `json:"jackpot"`
}

func GetRooms(c *gin.Context) {
	roomsMap := game.Manager.GetAllRooms()

	var rooms []RoomDTO

	// ==========================
	// EXISTING ROOMS
	// ==========================
	for stake, room := range roomsMap {

		room.Mutex.Lock()
		players := len(room.Players)
		status := room.State
		countdown := room.Countdown
		room.Mutex.Unlock()

		dto := RoomDTO{
			Stake:     stake,
			Players:   players,
			Win:       float64(players) * stake * 0.8,
			Status:    status,
			Countdown: countdown,
			Jackpot:   calculateJackpot(stake),
		}

		rooms = append(rooms, dto)
	}

	// ==========================
	// ENSURE DEFAULT ROOMS
	// ==========================
	defaultStakes := []float64{10, 20, 50, 100}

	for _, stake := range defaultStakes {
		if _, exists := roomsMap[stake]; !exists {

			room := game.Manager.GetRoom(stake)

			dto := RoomDTO{
				Stake:     stake,
				Players:   len(room.Players),
				Win:       0,
				Status:    "waiting",
				Countdown: 0,
				Jackpot:   calculateJackpot(stake),
			}

			rooms = append(rooms, dto)
		}
	}

	// ==========================
	// SORT (VERY NICE UX)
	// ==========================
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Stake < rooms[j].Stake
	})

	// ==========================
	// FINAL RESPONSE (ONLY ONCE)
	// ==========================
	c.JSON(200, rooms)
}

// ==========================
// JACKPOT LOGIC
// ==========================
func calculateJackpot(stake float64) float64 {
	return stake * 50
}