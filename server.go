package chancon

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"net"
	"sync"
	"time"
)

type Server struct {
	*observerManager
	*tlsManager

	connections     map[uuid.UUID]*connection
	connectionsLock *sync.RWMutex

	port int
}

func NewServer(port int) *Server {
	return &Server{
		observerManager: newObserverManager(),
		tlsManager:      newTlsManager(),
		connections:     map[uuid.UUID]*connection{},
		connectionsLock: &sync.RWMutex{},
		port:            port,
	}
}

func (s *Server) saveConnection(connection *connection) {
	s.connectionsLock.Lock()
	s.connections[connection.Id] = connection
	s.connectionsLock.Unlock()
}

func (s *Server) closeConnection(connection *connection) {
	_ = connection.Close()
	s.deleteConnection(connection)
	connection = nil
}

func (s *Server) deleteConnection(connection *connection) {
	s.connectionsLock.Lock()
	delete(s.connections, connection.Id)
	s.connectionsLock.Unlock()
}

func (s *Server) Broadcast(m *Message) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, connection := range s.connections {
		connection.Send(m)
	}
}
func (s *Server) On(channel string, callback ObserverCallback) func() {
	return s.observerManager.On(channel, callback)
}

func (s *Server) acceptNewConnection(conn net.Conn) {
	connection := NewConnection(conn, s.observerManager)

	s.saveConnection(connection)

	connection.On(IntroductionChannel, func(m *Message) error {
		if _, err := connection.Ping(); err != nil {
			s.closeConnection(connection)
			return err
		}

		introduction := struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		}{}
		if err := json.Unmarshal(m.Data, &introduction); err != nil {
			return err
		}

		connection.State = "connected"
		connection.Hostname = introduction.Name

		log.Infof("connection %s introduced as %s", connection.Id, connection.Hostname)

		return m.Reply([]byte(""))
	})

	connection.handle()

	s.closeConnection(connection)
}

func (s *Server) handleListener(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Error in accepting connection: %s", err.Error())
			continue
		}

		go func(conn net.Conn) {
			s.acceptNewConnection(conn)
		}(conn)
	}
}

func (s *Server) Start() error {
	config, err := s.loadTlsConfig()

	var listener net.Listener
	if err != nil {
		if err != ErrNoSslConfig {
			return err
		}

		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.port))
		if err != nil {
			return err
		}
	} else {
		listener, err = tls.Listen("tcp", fmt.Sprintf(":%d", s.port), config)
		if err != nil {
			return err
		}
	}

	log.Infof("Server is listening on port :%d", s.port)

	s.handleListener(listener)
	return nil
}
