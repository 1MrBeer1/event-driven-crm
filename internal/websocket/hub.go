package websocket

import (
	"context"
	"log/slog"

	goredis "github.com/redis/go-redis/v9"

	crmredis "github.com/1MrBeer1/event-driven-crm/internal/redis"
)

type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	logger     *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		logger:     logger,
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			return
		case client := <-h.register:
			h.clients[client] = struct{}{}
			h.logger.Info("websocket client connected", "clients", len(h.clients))
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.logger.Info("websocket client disconnected", "clients", len(h.clients))
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}
}

func (h *Hub) SubscribeRedis(ctx context.Context, client *goredis.Client) {
	pubsub := client.Subscribe(ctx, crmredis.NotificationsChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			h.broadcast <- []byte(msg.Payload)
		}
	}
}
