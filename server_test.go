package chancon

import (
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBroadcastSupportsConcurrentConnectionChanges(t *testing.T) {
	server := NewServer(0)
	message := &Message{
		Id: uuid.New(),
		Channel: Channel{
			Name: "test",
		},
		Date: time.Now(),
	}

	const workers = 8
	const iterations = 250

	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers + 1)

	for worker := 0; worker < workers; worker++ {
		go func() {
			defer waitGroup.Done()
			<-start

			for iteration := 0; iteration < iterations; iteration++ {
				connection := NewConnection(discardConnection{}, server.observerManager)
				server.saveConnection(connection)
				server.deleteConnection(connection)
			}
		}()
	}

	go func() {
		defer waitGroup.Done()
		<-start

		for iteration := 0; iteration < workers*iterations; iteration++ {
			server.Broadcast(message)
		}
	}()

	close(start)
	waitGroup.Wait()
}

type discardConnection struct{}

func (discardConnection) Read([]byte) (int, error)          { return 0, io.EOF }
func (discardConnection) Write(payload []byte) (int, error) { return len(payload), nil }
func (discardConnection) Close() error                      { return nil }
func (discardConnection) LocalAddr() net.Addr               { return testAddress("local") }
func (discardConnection) RemoteAddr() net.Addr              { return testAddress("remote") }
func (discardConnection) SetDeadline(time.Time) error       { return nil }
func (discardConnection) SetReadDeadline(time.Time) error   { return nil }
func (discardConnection) SetWriteDeadline(time.Time) error  { return nil }

type testAddress string

func (address testAddress) Network() string { return string(address) }
func (address testAddress) String() string  { return string(address) }
