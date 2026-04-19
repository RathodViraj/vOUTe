package ws

import (
	"context"
	"sync"
	"time"
	"voute/pkg/vote"

	"github.com/gorilla/websocket"
)

const (
	broadcastInterval = 3 * time.Second
	dirtyPollsKey     = "ws:dirty:polls"
)

type GlobalUpdate struct {
	Chnages []PollSnapshot `json:"changes"`
}

type PollSnapshot = vote.PollSnapshot
type OptionSnapshot = vote.OptionSnapshot

type Client struct {
	conn *websocket.Conn
	send chan GlobalUpdate
	hub  *Hub
}

func (c *Client) writePump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for update := range c.send {
		if err := c.conn.WriteJSON(update); err != nil {
			return
		}
	}
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	reader     PollReader
}

type PollReader interface {
	GetSnapshots(ctx context.Context, pollsIDs []string) ([]PollSnapshot, error)
	PopDirtyPolls(ctx context.Context) ([]string, error)
}

func NewHub(reader PollReader) *Hub {
	h := &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		reader:     reader,
	}

	go h.run()
	go h.broadcastLoop()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) broadcastLoop() {
	ticker := time.NewTicker(broadcastInterval)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		hasClients := len(h.clients) > 0
		h.mu.RUnlock()

		if !hasClients {
			time.Sleep(3 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		dirtyIDs, err := h.reader.PopDirtyPolls(ctx)
		if err != nil || len(dirtyIDs) == 0 {
			cancel()
			continue
		}

		snapshots, err := h.reader.GetSnapshots(ctx, dirtyIDs)
		cancel()
		if err != nil || len(snapshots) == 0 {
			continue
		}

		update := GlobalUpdate{Chnages: snapshots}

		h.mu.RLock()
		for client := range h.clients {
			select {
			case client.send <- update:
			default:
				// Slow client — drop it rather than blocking the broadcast.
				go func(c *Client) { h.unregister <- c }(client)
			}
		}
		h.mu.RUnlock()
	}
}

func (h *Hub) ServeWS(conn *websocket.Conn) {
	client := &Client{
		conn: conn,
		send: make(chan GlobalUpdate, 4),
		hub:  h,
	}
	h.register <- client
	go client.writePump()
}
