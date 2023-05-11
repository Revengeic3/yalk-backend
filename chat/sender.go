package chat

import (
	"log"
	"time"

	"yalk/chat/clients"
)

func (server *Server) Sender(c *clients.Client, ctx *EventContext) {
	defer ctx.WaitGroup.Done()

Run:
	for {
		select {
		case <-ctx.NotifyChannel:
			log.Println("Sender - got shutdown signal")
			break Run
		case payload := <-c.Msgs:
			err := clients.ClientWriteWithTimeout(ctx.Request.Context(), time.Second*5, ctx.Connection, payload)
			if err != nil {
				break Run
			}
		}
	}
}
