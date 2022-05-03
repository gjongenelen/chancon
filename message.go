package chancon

import (
	"github.com/google/uuid"
	"time"
)

type Message struct {
	Id      uuid.UUID `json:"id"`
	ReplyTo uuid.UUID `json:"reply_to"`
	Channel Channel   `json:"channel"`
	Data    []byte    `json:"data"`
	Origin  Host      `json:"origin"`
	Target  Host      `json:"target"`
	Date    time.Time `json:"date"`
}

func (m *Message) Reply(data []byte) error {
	return m.Channel.Connection.Send(&Message{
		Id:      uuid.New(),
		Channel: m.Channel,
		Origin:  m.Target,
		Target:  m.Origin,
		Date:    time.Now(),
		Data:    data,
		ReplyTo: m.Id,
	})
}
