package chat

import (
	"encoding/json"
	"log"
	"yalk/chat/events"
	"yalk/chat/models"
)

// func (server *Server) SendToAdmins(message *Message, payload []byte) {}

func (server *Server) Router() {
	for {
		select {
		case message := <-server.Channels.Messages:
			log.Printf("Router: Broadcast message received")

			serializedData, err := message.Serialize()
			if err != nil {
				log.Printf("Error serializing")
			}

			newRawEvent := events.RawEvent{UserID: message.UserID, Type: message.Type(), Data: serializedData}

			jsonEvent, err := json.Marshal(newRawEvent)
			if err != nil {
				log.Printf("Error serializing RawEvent")
			}

			server.SendToChat(message, jsonEvent)

		case rawEvent := <-server.Channels.Accounts:
			log.Printf("Router: Account event received")
			jsonEvent, err := json.Marshal(rawEvent)
			if err != nil {
				log.Printf("Error serializing RawEvent")
			}
			server.SendBack(rawEvent.UserID, jsonEvent)

		case rawEvent := <-server.Channels.Users:
			log.Printf("Router: User event received")
			jsonEvent, err := json.Marshal(rawEvent)
			if err != nil {
				log.Printf("Error serializing RawEvent")
			}
			server.SendAll(jsonEvent)
		}
	}
}

// Echoing to client is default behavior for error checking.

// TODO: Error checking
// func (server *Server) SendToChat(message *Message, payload []byte) {
// 	for _, user := range message.Chat.Users {
// 		if client, ok := server.Clients[user.ID]; ok {
// 			client.Msgs <- payload
// 		}
// 	}
// }

func (server *Server) SendToChat(message *models.Message, payload []byte) {
	// TODO: Move to server method
	chat := &models.Chat{ID: message.ChatID}
	user, err := chat.GetUsers(server.Db)
	if err != nil {
		log.Printf("Error getting chat users")
		return
	}
	for _, user := range user {
		if client, ok := server.Clients[user.ID]; ok {
			client.Msgs <- payload
		}
	}
}

func (server *Server) SendBack(id uint, payload []byte) {
	if client, ok := server.Clients[id]; ok {
		client.Msgs <- payload
	}
}

func (server *Server) SendAll(payload []byte) {
	for _, client := range server.Clients {
		client.Msgs <- payload
	}
}
