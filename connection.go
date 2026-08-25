package chancon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"net"
	"sync"
	"time"
)

var PingChannel = "*ping"

func NewConnection(conn net.Conn, observerManager *observerManager) *connection {
	return &connection{
		privateObserver: newObserverManager(),
		publicObserver:  observerManager,
		Id:              uuid.New(),
		State:           "initiated",
		conn:            conn,
		writeLock:       &sync.Mutex{},
	}
}

type connection struct {
	privateObserver *observerManager
	publicObserver  *observerManager

	Id       uuid.UUID
	State    string
	Hostname string

	conn     net.Conn
	lastPing time.Time

	writeLock *sync.Mutex
}

func (c *connection) handleMessage(message *Message) {
	c.privateObserver.handle(message)

	c.publicObserver.handle(message)
}

func (c *connection) On(channel string, callback ObserverCallback) func() {
	return c.privateObserver.On(channel, callback)
}

func (c *connection) Ping() (time.Duration, error) {
	pingMessage := &Message{
		Id: uuid.New(),
		Channel: Channel{
			Name: PingChannel,
		},
		Date: time.Now(),
	}

	reply, err := c.SendAndWaitForReply(pingMessage)
	if err != nil {
		return -1, err
	}

	return reply.Date.Sub(pingMessage.Date), nil
}

func (c *connection) Close() error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	return c.conn.Close()
}

func (c *connection) SendAndWaitForReply(message *Message) (*Message, error) {
	return c.SendAndWaitForReplyWithTimeout(message, 10*time.Second)
}

func (c *connection) SendAndWaitForReplyWithTimeout(message *Message, timeout time.Duration) (*Message, error) {
	reply := make(chan *Message, 1)
	unsub := c.On(message.Channel.Name, func(m *Message) error {
		if m.ReplyTo == message.Id {
			select {
			case reply <- m:
			default:
			}
		}

		return nil
	})
	defer unsub()

	if err := c.Send(message); err != nil {
		return nil, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case replyMessage := <-reply:
		return replyMessage, nil
	case <-timer.C:
		return nil, ErrTimedOutWaitingForReply
	}
}

func (c *connection) Send(message *Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	payload = append(payload, []byte("\n")...)

	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	count, err := c.conn.Write(payload)
	if err != nil {
		_ = c.conn.Close()
		return err
	}
	if count != len(payload) {
		return errors.New("content-length other than bytes written")
	}

	return nil
}

func (c *connection) handle() {
	reader := bufio.NewReader(c.conn)

	defer func() {
		c.Close()
	}()

	for {
		payload, err := readReader(reader)
		if err != nil {
			return
		}
		if len(payload) == 0 {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		message := &Message{}
		err = json.Unmarshal(payload, message)
		if err != nil {
			log.Errorf("dropped message: %s", err.Error())
			log.Debug(string(payload))
			continue
		}
		message.Channel.Connection = c

		c.handleMessage(message)
	}

}

func readReader(reader *bufio.Reader) ([]byte, error) {

	var buffer bytes.Buffer

	if reader == nil {
		return []byte{}, nil
	}
	more := true
	for more {
		var ba []byte
		var err error

		ba, more, err = reader.ReadLine()
		if err != nil && err.Error() == "EOF" {
			return nil, err
		}
		buffer.Write(ba)
	}

	return buffer.Bytes(), nil
}
