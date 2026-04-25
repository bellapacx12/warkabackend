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
	Send   chan []byte
    Username string // ✅ required
	Connected bool
	LastSeen  time.Time
}

type Room struct {
	Stake     float64
	Players   map[int]*Player
	UsedCards map[int]bool
	Mutex     sync.Mutex
	Cards map[int][][]any // 🔥 ADD THIS
    UserCard map[int] *models.BingoCard
	State     string
	Countdown int

	Numbers []int
	Called  []int
}

const MinPlayers = 1

// ==========================
// SAFE SEND
// ==========================
func (p *Player) SendJSON(event string, data interface{}) {
	if !p.Connected {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Println("🔥 Recovered send panic for:", p.UserID)
		}
	}()

	msg := map[string]interface{}{
		"type": event,
		"data": data,
	}

	bytes, _ := json.Marshal(msg)

	select {
	case p.Send <- bytes:
	default:
		log.Println("⚠️ Dropping message for:", p.UserID)
	}
}

// ==========================
// INIT
// ==========================
func NewRoom(stake float64) *Room {
	room := &Room{
		Stake:     stake,
		Players:   make(map[int]*Player),
		Cards:     make(map[int][][]any),
		UserCard:     make(map[int]*models.BingoCard),  // 🔥 IMPORTANT
		UsedCards: make(map[int]bool),
		State:     "waiting",
	}

	go room.CleanupDisconnected()
	return room
}

// ==========================
// GAME STATE SYNC
// ==========================
func (r *Room) SendGameState(p *Player) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	state := map[string]interface{}{
		"state":     r.State,
		"countdown": r.Countdown,
		"called":    r.Called,
	}

	p.SendJSON("init", state)

	if p.Card != nil {
		p.SendJSON("card", p.Card)
	}
}

// ==========================
// HELPERS
// ==========================
func (r *Room) getActivePlayers() []*Player {
	var players []*Player
	for _, p := range r.Players {
		if p.Connected {
			players = append(players, p)
		}
	}
	return players
}

func (r *Room) enoughPlayers() bool {
	return len(r.getActivePlayers()) >= MinPlayers
}

func (r *Room) allPlayersHaveCards() bool {
	for _, p := range r.Players {
		if p.Connected && p.Card == nil {
			return false
		}
	}
	return true
}

// ==========================
// PLAYER JOIN / RECONNECT
// ==========================
func (r *Room) AddPlayer(p *Player) {
	r.Mutex.Lock()

	if existing, ok := r.Players[p.UserID]; ok {
		existing.Conn = p.Conn
		existing.Send = p.Send
		existing.Connected = true
		existing.LastSeen = time.Now()

		r.Mutex.Unlock()

		log.Println("🔄 Reconnected:", p.UserID)

		r.SendAvailableCards(existing)
		r.SendTakenCards(existing)
		r.SendGameState(existing)

		go BroadcastLobby()
		return
	}

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
	r.SendGameState(p)

	go BroadcastLobby()

	if state == "waiting" && r.enoughPlayers() && r.allPlayersHaveCards() {
		go r.StartCountdown()
	}
}

// ==========================
// CARD SELECTION
// ==========================
func convertToGrid(card *models.BingoCard) [][]any {
	grid := make([][]any, 5)

	for i := 0; i < 5; i++ {
		grid[i] = make([]any, 5)
	}

	for i := 0; i < 5; i++ {
		grid[i][0] = valueOrNil(card.B[i])
		grid[i][1] = valueOrNil(card.I[i])
		grid[i][2] = valueOrNil(card.N[i])
		grid[i][3] = valueOrNil(card.G[i])
		grid[i][4] = valueOrNil(card.O[i])
	}

	// ✅ FREE SPACE
	grid[2][2] = "FREE"

	return grid
}

func valueOrNil(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
func (r *Room) HandleSelectCard(userID int, cardID int) {
	r.Mutex.Lock()

	if r.UsedCards[cardID] {
		r.Mutex.Unlock()
		return
	}

	player, ok := r.Players[userID]
	if !ok {
		r.Mutex.Unlock()
		return
	}

	if player.Card != nil && player.Card.CardID == cardID {
		r.Mutex.Unlock()
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
		r.Mutex.Unlock()
		return
	}
	

	// ✅ SAVE CARD
	player.Card = selected
	r.UserCard[userID] = selected

	r.UsedCards[cardID] = true
    
	// ✅ CONVERT TO GRID
	grid := convertToGrid(selected)

	// ✅ STORE GRID FOR RECONNECT
	r.Cards[userID] = grid

	r.Mutex.Unlock()

	// ✅ SEND CORRECT FORMAT
	player.SendJSON("card_selected", map[string]interface{}{
		"card_id": cardID,
	})

	player.SendJSON("card", map[string]interface{}{
		"grid": grid,
	})

	r.Broadcast("card_taken", cardID)

	log.Printf("✅ Player %d took card %d\n", userID, cardID)

	if r.State == "waiting" && r.enoughPlayers() && r.allPlayersHaveCards() {
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
				log.Println("🗑 Removed:", id)
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
	r.Mutex.Lock()
	var players []*Player
	for _, p := range r.Players {
		if p.Connected {
			players = append(players, p)
		}
	}
	r.Mutex.Unlock()

	for _, p := range players {
		p.SendJSON(event, data)
	}
}

// ==========================
// GAME FLOW
// ==========================
func (r *Room) StartCountdown() {
	r.Mutex.Lock()
	if r.State != "waiting" || !r.enoughPlayers() || !r.allPlayersHaveCards() {
		r.Mutex.Unlock()
		return
	}

	r.State = "countdown"
	r.Mutex.Unlock()

	for i := 10; i > 0; i-- {
		r.Mutex.Lock()

		if !r.enoughPlayers() || !r.allPlayersHaveCards() {
			r.State = "waiting"
			r.Mutex.Unlock()
			return
		}

		r.Countdown = i
		r.Mutex.Unlock()

		r.Broadcast("countdown", i)
		time.Sleep(time.Second)
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
// CARD SYSTEM
// ==========================
func (r *Room) SendAvailableCards(p *Player) {
	var available []map[string]int

	r.Mutex.Lock()
	for _, c := range storage.Cards {
		if !r.UsedCards[c.CardID] {
			available = append(available, map[string]int{
				"card_id": c.CardID,
			})
		}
	}
	r.Mutex.Unlock()

	p.SendJSON("cards", available)
}

func (r *Room) SendTakenCards(p *Player) {
	taken := []int{}

	r.Mutex.Lock()
	for id := range r.UsedCards {
		taken = append(taken, id)
	}
	r.Mutex.Unlock()

	p.SendJSON("taken_cards", taken)
}

// ==========================
// WRITE PUMP
// ==========================
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
// ==========================
// RECONNECT PLAYER
// ==========================
func (r *Room) ReconnectPlayer(player *Player) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	oldPlayer, exists := r.Players[player.UserID]

	if exists {
		log.Printf("♻️ Replacing connection for user %d\n", player.UserID)

		// ❌ close old connection safely
		if oldPlayer.Conn != nil {
			oldPlayer.Conn.Close()
		}
	}

	// ✅ replace with new connection
	r.Players[player.UserID] = player
}
// ==========================
// GET PLAYER CARD
// ==========================
func (r *Room) GetPlayerCard(userID int) *models.BingoCard {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	card, ok := r.UserCard[userID]
	if !ok {
		return nil
	}

	return card
}
func (r *Room) ResetRound() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	log.Println("🔄 Resetting round...")

	// reset game state
	r.State = "waiting"
	r.Countdown = 0
	r.Numbers = nil
	r.Called = []int{}

	// clear cards
	r.UsedCards = make(map[int]bool)
	r.Cards = make(map[int][][]any)

	// clear player cards
	for _, p := range r.Players {
		p.Card = nil
	}

	// notify players
	go r.Broadcast("round_reset", "new round started")
}