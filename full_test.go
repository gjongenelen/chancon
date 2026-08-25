package chancon

import (
	"bytes"
	"github.com/google/uuid"
	"net"
	"testing"
	"time"
)

func TestFullSuite(t *testing.T) {
	port := availableTCPPort(t)
	server := NewServer(port)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Start()
	}()

	testMessage := []byte("test_msg")
	testReplyMessage := []byte("reply_test_msg")

	msgChan := make(chan []byte, 1)
	server.On("test", func(m *Message) error {
		msgChan <- m.Data
		return m.Reply(testReplyMessage)
	})

	client := NewClient("localhost", port)
	connected := make(chan struct{}, 1)
	client.On("*connected", func(*Message) error {
		connected <- struct{}{}
		return nil
	})
	go func() {
		_ = client.Connect()
	}()

	select {
	case err := <-serverErrors:
		t.Fatalf("server failed before the client connected: %v", err)
	case <-connected:
	case <-time.After(5 * time.Second):
		t.Fatal("client did not connect within 5 seconds")
	}
	t.Cleanup(func() { _ = client.Close() })

	reply, err := client.SendAndWaitForReplyWithTimeout(&Message{
		Id: uuid.New(),
		Channel: Channel{
			Name: "test",
		},
		Data: testMessage,
	}, 5*time.Second)
	if err != nil {
		t.Fatal("got error on client-send: " + err.Error())
	}
	if !bytes.Equal(reply.Data, testReplyMessage) {
		t.Errorf("reply data = %q, want %q", reply.Data, testReplyMessage)
	}
	select {
	case msg := <-msgChan:
		if !bytes.Equal(msg, testMessage) {
			t.Errorf("message data = %q, want %q", msg, testMessage)
		}
	case <-time.After(1 * time.Second):
		t.Error("no msg after 1 sec")
	}
}

func availableTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve TCP port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release TCP port: %v", err)
	}

	return port
}
