package chancon

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSendAndWaitRegistersObserverBeforeSending(t *testing.T) {
	networkConnection := &immediateReplyConnection{}
	connection := NewConnection(networkConnection, newObserverManager())
	networkConnection.connection = connection

	request := &Message{
		Id: uuid.New(),
		Channel: Channel{
			Name: "test",
		},
		Date: time.Now(),
	}
	reply, err := connection.SendAndWaitForReplyWithTimeout(request, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("SendAndWaitForReplyWithTimeout() error = %v", err)
	}
	if reply.ReplyTo != request.Id {
		t.Fatalf("reply_to = %s, want %s", reply.ReplyTo, request.Id)
	}
}

func TestSendReturnsAfterWriteFailure(t *testing.T) {
	expectedError := errors.New("write failed")
	networkConnection := &failingWriteConnection{
		writeError: expectedError,
		closed:     make(chan struct{}),
	}
	connection := NewConnection(networkConnection, newObserverManager())

	result := make(chan error, 1)
	go func() {
		result <- connection.Send(&Message{
			Id: uuid.New(),
			Channel: Channel{
				Name: "test",
			},
			Date: time.Now(),
		})
	}()

	select {
	case err := <-result:
		if !errors.Is(err, expectedError) {
			t.Fatalf("Send() error = %v, want %v", err, expectedError)
		}
	case <-time.After(time.Second):
		t.Fatal("Send() deadlocked after the connection returned a write error")
	}

	select {
	case <-networkConnection.closed:
	case <-time.After(time.Second):
		t.Fatal("Send() did not close the connection after a write error")
	}
}

type failingWriteConnection struct {
	writeError error
	closed     chan struct{}
}

func (*failingWriteConnection) Read([]byte) (int, error) { return 0, errors.New("not implemented") }
func (connection *failingWriteConnection) Write([]byte) (int, error) {
	return 0, connection.writeError
}
func (connection *failingWriteConnection) Close() error {
	select {
	case <-connection.closed:
	default:
		close(connection.closed)
	}
	return nil
}
func (*failingWriteConnection) LocalAddr() net.Addr              { return testAddress("local") }
func (*failingWriteConnection) RemoteAddr() net.Addr             { return testAddress("remote") }
func (*failingWriteConnection) SetDeadline(time.Time) error      { return nil }
func (*failingWriteConnection) SetReadDeadline(time.Time) error  { return nil }
func (*failingWriteConnection) SetWriteDeadline(time.Time) error { return nil }

type immediateReplyConnection struct {
	connection *connection
}

func (*immediateReplyConnection) Read([]byte) (int, error) { return 0, errors.New("not implemented") }
func (networkConnection *immediateReplyConnection) Write(payload []byte) (int, error) {
	request := new(Message)
	if err := json.Unmarshal(bytes.TrimSpace(payload), request); err != nil {
		return 0, err
	}
	networkConnection.connection.handleMessage(&Message{
		Id:      uuid.New(),
		ReplyTo: request.Id,
		Channel: request.Channel,
		Date:    time.Now(),
	})
	return len(payload), nil
}
func (*immediateReplyConnection) Close() error                     { return nil }
func (*immediateReplyConnection) LocalAddr() net.Addr              { return testAddress("local") }
func (*immediateReplyConnection) RemoteAddr() net.Addr             { return testAddress("remote") }
func (*immediateReplyConnection) SetDeadline(time.Time) error      { return nil }
func (*immediateReplyConnection) SetReadDeadline(time.Time) error  { return nil }
func (*immediateReplyConnection) SetWriteDeadline(time.Time) error { return nil }
