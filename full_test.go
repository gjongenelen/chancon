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
	testReplyMsg := []byte("reply_test_msg")

	msgChan := make(chan []byte, 1)
	server.On("test", func(m *Message) error {
		msgChan <- m.Data
		return m.Reply(append(testReplyMsg))
	})

	client := NewClient("localhost", 1234)
	err := client.Connect()
	if err != nil {
		t.Fatal("got error on client-connect: " + err.Error())
	}

	reply, err := client.SendAndWaitForReplyWithTimeout(&Message{
		Id: uuid.New(),
		Channel: Channel{
			Name: "test",
		},
		Data: append(testMsg),
	}, 5*time.Second)
	if err != nil {
		t.Fatal("got error on client-send: " + err.Error())
	}
	if len(reply.Data) != len(testReplyMsg) {
		t.Error("invalid reply-data received")
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
