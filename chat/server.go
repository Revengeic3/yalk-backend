package chat

import (
	"sync"
	"time"
	"yalk/chat/clients"
	"yalk/chat/events"

	"github.com/lib/pq"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
	"nhooyr.io/websocket"
)

type ChatServer interface {
	RegisterClient(*websocket.Conn, string)
	SendMessage(*events.Event)
	SendMessageToAll(*events.Event)
	Sender(*clients.Client, *events.EventContext)
	Receiver(*events.EventContext)
	HandlePayload([]byte)
}

// TODO: db
func NewServer(bufferLenght uint, db *gorm.DB) *Server {

	sendLimiter := rate.NewLimiter(rate.Every(time.Millisecond*100), 8)
	clientsMap := make(map[uint]*clients.Client)
	messageChannels := events.MakeEventChannels()

	chatServer := &Server{
		SendLimiter:          sendLimiter,
		Clients:              clientsMap,
		ClientsMessageBuffer: bufferLenght,
		Channels:             messageChannels,
		Db:                   db,
	}

	return chatServer
}

type Server struct {
	SendLimiter          *rate.Limiter
	Clients              map[uint]*clients.Client
	ClientsMu            sync.Mutex
	ClientsMessageBuffer uint
	Channels             *events.EventChannels
	Db                   *gorm.DB
}

func (server *Server) RegisterClient(conn *websocket.Conn, id uint) *clients.Client {
	messageChan := make(chan []byte, server.ClientsMessageBuffer)

	client := &clients.Client{
		ID:   id,
		Msgs: messageChan,
		CloseSlow: func() {
			conn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}
	server.ClientsMu.Lock()
	server.Clients[id] = client
	server.ClientsMu.Unlock()
	return client
}

type BinaryPayload struct {
	Success bool   `json:"success"`
	Origin  string `json:"origin,omitempty"`
	Event   string `json:"event"`
	// Data    []byte `json:"data,omitempty"`
}

type ChatList struct {
	ID           uint           `gorm:"id;primaryKey"`
	Name         string         `gorm:"name" json:"name"`
	Users        pq.StringArray `gorm:"type:text[];users" json:"users"`
	CreatedBy    string         `gorm:"createdBy" json:"createdBy"`
	CreationDate time.Time      `gorm:"creationDate" json:"creationDate"`
}
