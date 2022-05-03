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
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), 5*time.Second)
	if err != nil {
		return err
	}

	tlsConfig, err := c.loadTlsConfig()
	if err == nil {
		conn = tls.Client(conn, tlsConfig)
	}

	c.connection = NewConnection(conn, c.observerManager)
	c.On(PingChannel, func(m *Message) error {
		return m.Reply([]byte(""))
	})

	go func() {
		c.connection.handle()
		log.Error("connection lost")
		err = errors.New("")
		for err != nil {
			err = c.Connect()
			time.Sleep(20 * time.Millisecond)
		}
	}()

	err = c.introduce()
	if err != nil {
		return err
	}
	log.Info("Introduced myself")
	c.observerManager.handle(&Message{
		Channel: Channel{
			Name: "*connected",
		},
	})
	return nil
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
