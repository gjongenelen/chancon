package chancon

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestFullSuite(t *testing.T) {

	server := NewServer(1234)
	go func() {
		err := server.Start()
		if err != nil {
			t.Error("got error on server-start: " + err.Error())
		}
	}()

	testMsg := []byte("test_msg")

	msgChan := make(chan []byte)
	server.On("test", func(m *Message) error {
		msgChan <- m.Data
		return nil
	})

	client := NewClient("localhost", 1234)
	err := client.Connect()
	if err != nil {
		t.Fatal("got error on client-connect: " + err.Error())
	}

	err = client.Send(&Message{
		Id: uuid.New(),
		Channel: Channel{
			Name: "test",
		},
		Data: append(testMsg),
	})
	if err != nil {
		t.Fatal("got error on client-send: " + err.Error())
	}

	select {
	case msg := <-msgChan:
		if len(testMsg) != len(msg) {
			t.Error("invalid data received")
		}
		return
	case <-time.After(1 * time.Second):
		t.Error("no msg after 1 sec")
	}
}
