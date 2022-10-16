package chancon

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"net"
	"os"
	"time"
)

type Client struct {
	*observerManager
	*tlsManager

	host string
	port int
	*connection
}

func NewClient(host string, port int) *Client {
	return &Client{
		observerManager: newObserverManager(),
		tlsManager:      newTlsManager(),
		host:            host,
		port:            port,
	}
}

func (c *Client) Connect() error {
	for {
		c.connect()

		time.Sleep(1 * time.Second)
	}
}

func (c *Client) connect() error {
	defer c.observerManager.handle(&Message{
		Channel: Channel{
			Name: "*disconnected",
		},
	})

	log.Debug("Connecting")
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), 2*time.Second)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return err
	}

	tlsConfig, err := c.loadTlsConfig()
	if err == nil {
		conn = tls.Client(conn, tlsConfig)
	}

	c.connection = NewConnection(conn, c.observerManager)
	c.connection.lastPing = time.Now()
	unsub := c.publicObserver.On(PingChannel, func(m *Message) error {
		c.connection.lastPing = time.Now()
		return m.Reply([]byte(""))
	})
	defer unsub()

	errChan := make(chan error)
	go func() {
		c.connection.handle()
		errChan <- errors.New("connection closed")
	}()

	err = c.introduce()
	if err != nil {
		c.Close()
		return err
	}
	log.Info("Introduced myself")
	c.observerManager.handle(&Message{
		Channel: Channel{
			Name: "*connected",
		},
	})

	return <-errChan
}

func (c *Client) introduce() error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	dump, err := json.Marshal(&Introduction{
		Name: hostname,
		Date: time.Now(),
	})
	if err != nil {
		return err
	}

	_, err = c.connection.SendAndWaitForReplyWithTimeout(&Message{
		Id:   uuid.New(),
		Data: dump,
		Channel: Channel{
			Name: IntroductionChannel,
		},
		Date: time.Now(),
	}, 5*time.Second)

	return err
}

func (c *Client) On(channel string, callback ObserverCallback) func() {
	return c.observerManager.On(channel, callback)
}
