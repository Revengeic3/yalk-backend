package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"yalk/cattp"
	"yalk/chat"
	"yalk/chat/clients"
	"yalk/logger"

	"gorm.io/gorm"
	"nhooyr.io/websocket"
)

func startHttpServer(conf cattp.Config, chatServer *chat.Server) error {
	router := cattp.New(chatServer)
	router.HandleFunc("/ws", connectHandle)

	err := router.Listen(&conf)
	if err != nil {
		return err
	}

	log.Println("HTTP Server succesfully started") // TODO: Move back in main func
	return nil
}

// TODO: Enum log events and colors 'info' 'warning 'error'
var connectHandle = cattp.HandlerFunc[*chat.Server](func(w http.ResponseWriter, r *http.Request, server *chat.Server) {

	logger.Info("WEBSOCK", fmt.Sprintf("Requested WebSocket - %s", r.RemoteAddr))

	conn, err := upgradeHttpRequest(w, r)
	if err != nil {
		logger.Err("WEBSOCK", fmt.Sprintf("Can't start accepting - %s", r.RemoteAddr))
		w.WriteHeader(http.StatusInternalServerError)
		r.Body.Close()
		return
	}

	defer conn.Close(websocket.StatusNormalClosure, "Client disconnected")

	// Todo: Use profile instead of User ID?
	client := server.RegisterClient(conn, 1)

	notify := make(chan bool)

	var wg sync.WaitGroup

	// Ping ticker which must be switched to the literally already fully made
	// methods on the connection.
	var ticker = time.NewTicker(time.Second * time.Duration(100000))

	channelsContext := &chat.EventContext{
		NotifyChannel: notify,
		PingTicket:    ticker,
		WaitGroup:     &wg,
		Request:       r,
		Connection:    conn,
	}

	wg.Add(1)
	go server.Receiver(client.ID, channelsContext)

	wg.Add(1)
	go server.Sender(client, channelsContext)

	initalPayload, err := makeInitialPayload(server.Db)
	if err != nil {
		logger.Err("CORE", "Error marshalling payload")
		return
	}

	if clients.ClientWriteWithTimeout(r.Context(), time.Second*5, conn, initalPayload); err != nil {
		logger.Info("CLIENT", "Timeout Initial Payload")
		return
	}

	logger.Info("CLIENT", fmt.Sprintf("Full data sent to ID: %v", "test"))

	wg.Wait()

	go func() {
		channelsContext.NotifyChannel <- true
		server.ClientsMu.Lock()
		delete(server.Clients, client.ID)
		server.ClientsMu.Unlock()
		// onlineTick := time.NewTicker(time.Second * 10)
		// <-onlineTick.C
	}()
	logger.Info("CORE", fmt.Sprintf("Closed client ID %s", "test"))
})

func makeInitialPayload(db *gorm.DB) ([]byte, error) {
	// TODO: Check for duplicate data everywhere
	var user = &chat.User{ID: 1}
	_, err := user.GetInfo(db)
	if err != nil {
		return nil, err
	}

	var chats *[]chat.Chat
	tx := db.Joins("left join chat_users on chat_users.chat_id=chats.id").
		Where("chat_users.user_id = ?", user.ID). // TODO: Check for public chat
		Preload("Messages").
		Preload("Messages.User").
		Preload("ChatType").
		Find(&chats)
	if tx.Error != nil {
		return nil, tx.Error
	}

	temp := struct {
		User  *chat.User   `json:"user,omitempty"`
		Chats *[]chat.Chat `json:"chats,omitempty"`
	}{user, chats}

	jsonPayload, err := json.Marshal(&temp)
	if err != nil {
		return nil, err
	}

	newRawEvent := &chat.RawEvent{Type: "initial", Data: jsonPayload}

	jsonEvent, err := newRawEvent.Serialize()
	fmt.Println(string(jsonEvent))
	if err != nil {
		return nil, err
	}

	return jsonEvent, nil
}
