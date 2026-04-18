package game

import (
	"bingo-backend/models"
	"bingo-backend/storage"
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	UserID int
	Conn   *websocket.Conn
	Card   *models.BingoCard
	Send   chan []byte // ✅ NEW


	Connected bool
	LastSeen  time.Time
}

type Room struct {
	Stake     float64
	Players   map[int]*Player
	UsedCards map[int]bool
	Mutex     sync.Mutex

	State     string // "waiting", "countdown", "playing"
	Countdown int

	Numbers []int
	Called  []int
}

const MinPlayers = 1

// ==========================
// INIT
// ==========================
func NewRoom(stake float64) *Room {
	room := &Room{
		Stake:     stake,
		Players:   make(map[int]*Player),
		UsedCards: make(map[int]bool),
		State:     "waiting",
	}

	go room.CleanupDisconnected()
	return room
}

// ==========================
// HELPERS
// ==========================
func (r *Room) getActivePlayers() []*Player {
	players := []*Player{}
	for _, p := range r.Players {
		if p.Connected {
			players = append(players, p)
		}
	}
	return players
}

func (r *Room) allPlayersHaveCards() bool {
	for _, p := range r.Players {
		if p.Connected && p.Card == nil {
			return false
		}
	}
	return len(r.getActivePlayers()) >= MinPlayers
}

// ==========================
// PLAYER JOIN / RECONNECT
// ==========================
func (r *Room) AddPlayer(p *Player) {
	r.Mutex.Lock()

	// 🔄 RECONNECT
	if existing, ok := r.Players[p.UserID]; ok {
		existing.Conn = p.Conn
		existing.Connected = true
		existing.LastSeen = time.Now()

		r.Mutex.Unlock()

		log.Println("🔄 Player reconnected:", p.UserID)

		r.SendAvailableCards(existing)
		r.SendTakenCards(existing)

		if existing.Card != nil {
			existing.Conn.WriteJSON(map[string]interface{}{
				"type": "card_selected",
				"data": existing.Card,
			})
		}

		go BroadcastLobby()
		return
	}

	// 🆕 NEW PLAYER
	p.Connected = true
	p.LastSeen = time.Now()

	r.Players[p.UserID] = p

	count := len(r.Players)
	state := r.State

	r.Mutex.Unlock()

	log.Printf("Player %d joined room %.0f\n", p.UserID, r.Stake)

	r.Broadcast("players", count)
	r.SendAvailableCards(p)
	r.SendTakenCards(p)

	go BroadcastLobby()

	// 🚀 ONLY START IF READY
	if count >= MinPlayers && state == "waiting" && r.allPlayersHaveCards() {
		log.Println("🚀 Starting countdown...")
		go r.StartCountdown()
	}
}

// ==========================
// CARD SELECTION
// ==========================
func (r *Room) HandleSelectCard(userID int, cardID int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if r.UsedCards[cardID] {
		return
	}

	player, ok := r.Players[userID]
	if !ok {
		return
	}

	var selected *models.BingoCard

	for _, c := range storage.Cards {
		if c.CardID == cardID {
			tmp := c
			selected = &tmp
			break
		}
	}

	if selected == nil {
		return
	}

	player.Card = selected
	r.UsedCards[cardID] = true

	player.Conn.WriteJSON(map[string]interface{}{
		"type": "card_selected",
		"data": selected,
	})

	go r.Broadcast("card_taken", cardID)

	log.Printf("✅ Player %d took card %d\n", userID, cardID)

	// 🔥 CHECK IF ALL READY → START
	if r.State == "waiting" && r.allPlayersHaveCards() {
		go r.StartCountdown()
	}

	go BroadcastLobby()
}

// ==========================
// DISCONNECT
// ==========================
func (r *Room) MarkDisconnected(userID int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	if p, ok := r.Players[userID]; ok {
		p.Connected = false
		p.LastSeen = time.Now()
	}
}

// ==========================
// CLEANUP
// ==========================
func (r *Room) CleanupDisconnected() {
	for {
		time.Sleep(30 * time.Second)

		r.Mutex.Lock()

		for id, p := range r.Players {
			if !p.Connected && time.Since(p.LastSeen) > 60*time.Second {

				if p.Card != nil {
					delete(r.UsedCards, p.Card.CardID)
				}

				delete(r.Players, id)
				log.Println("🗑 Removed inactive:", id)
			}
		}

		r.Mutex.Unlock()
		go BroadcastLobby()
	}
}

// ==========================
// BROADCAST
// ==========================
func (r *Room) Broadcast(event string, data interface{}) {
	msg := map[string]interface{}{
		"type": event,
		"data": data,
	}

	bytes, _ := json.Marshal(msg)

	r.Mutex.Lock()
	players := make([]*Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, p)
	}
	r.Mutex.Unlock()

	for _, p := range players {
		if p.Conn == nil || !p.Connected {
			continue
		}

		p.Conn.WriteMessage(websocket.TextMessage, bytes)
	}
}

// ==========================
// GAME FLOW
// ==========================
func (r *Room) StartCountdown() {
	r.Mutex.Lock()
	if r.State != "waiting" || !r.allPlayersHaveCards() {
		r.Mutex.Unlock()
		return
	}

	r.State = "countdown"
	r.Mutex.Unlock()

	log.Println("⏳ Countdown started")

	for i := 10; i > 0; i-- {

		r.Mutex.Lock()

		// ❌ STOP if someone lost card / left
		if !r.allPlayersHaveCards() {
			r.State = "waiting"
			r.Mutex.Unlock()

			log.Println("⛔ Countdown cancelled")
			go BroadcastLobby()
			return
		}

		r.Countdown = i
		r.Mutex.Unlock()

		r.Broadcast("countdown", i)
		go BroadcastLobby()

		time.Sleep(1 * time.Second)
	}

	r.StartGame()
}

func (r *Room) StartGame() {
	r.Mutex.Lock()
	r.State = "playing"
	r.Numbers = generateNumbers()
	r.Called = []int{}
	r.Mutex.Unlock()

	r.Broadcast("start", "Game started!")
	go BroadcastLobby()

	go r.CallNumbers()
}

func generateNumbers() []int {
	nums := make([]int, 75)
	for i := range nums {
		nums[i] = i + 1
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(nums), func(i, j int) {
		nums[i], nums[j] = nums[j], nums[i]
	})

	return nums
}

func (r *Room) CallNumbers() {
	for _, n := range r.Numbers {

		r.Mutex.Lock()
		if r.State != "playing" {
			r.Mutex.Unlock()
			return
		}
		r.Called = append(r.Called, n)
		r.Mutex.Unlock()

		r.Broadcast("number", n)
		time.Sleep(2 * time.Second)
	}
}

// ==========================
// UTILS
// ==========================
func (r *Room) GetWinAmount() float64 {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	return float64(len(r.getActivePlayers())) * r.Stake * 0.8
}
// ==========================
// CARD SYSTEM
// ==========================
func (r *Room) SendAvailableCards(p *Player) {
	available := []map[string]int{}

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for _, c := range storage.Cards {
		if !r.UsedCards[c.CardID] {
			available = append(available, map[string]int{
				"card_id": c.CardID,
			})
		}
	}

	if p.Conn != nil {
		p.Conn.WriteJSON(map[string]interface{}{
			"type": "cards",
			"data": available,
		})
	}
}

func (r *Room) SendTakenCards(p *Player) {
	taken := []int{}

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	for id := range r.UsedCards {
		taken = append(taken, id)
	}

	if p.Conn != nil {
		p.Conn.WriteJSON(map[string]interface{}{
			"type": "taken_cards",
			"data": taken,
		})
	}
}
func (p *Player) WritePump() {
	defer p.Conn.Close()

	for msg := range p.Send {
		err := p.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Write error:", err)
			return
		}
	}
}
func (p *Player) ReadPump(onMessage func([]byte)) {
	defer p.Conn.Close()

	for {
		_, msg, err := p.Conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		onMessage(msg)
	}
}