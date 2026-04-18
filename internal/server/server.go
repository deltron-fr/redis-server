package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/deltron-fr/redis-server/internal/parser"
)

type Server struct {
	Store     map[string]ValueStore
	ListStore map[string][]string
	Commands  map[string]CommandHandler
	Mu        sync.RWMutex
}

type ValueStore struct {
	Value  string
	Expiry *time.Time
}

func NewServer() *Server {
	s := &Server{
		Store:     make(map[string]ValueStore),
		ListStore: make(map[string][]string),
	}

	s.Commands = map[string]CommandHandler{
		"ECHO":   s.echoHandler,
		"PING":   s.pingHandler,
		"SET":    s.setHandler,
		"GET":    s.getHandler,
		"RPUSH":  s.rPushHandler,
		"LPUSH":  s.lPushHandler,
		"LRANGE": s.lRangeHandler,
		"LLEN":   s.lLenHandler,
		"LPOP":   s.lPopHandler,
	}

	return s
}

// HandleConn reads RESP requests off the connection, dispatches each to the
// registered command handler, and writes the response back.
func (s *Server) HandleConn(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Println("read error:", err)
			}
			return
		}

		args, err := parser.Parse(buf[:n])
		if err != nil {
			writeErr(conn, err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		name := strings.ToUpper(args[0])
		handler, ok := s.Commands[name]
		if !ok {
			writeErr(conn, fmt.Errorf("unknown command %q", name))
			continue
		}

		resp, err := handler(Command{Args: args[1:]})
		if err != nil {
			writeErr(conn, err)
			continue
		}

		if _, err := conn.Write([]byte(resp)); err != nil {
			return
		}
	}
}

func writeErr(conn net.Conn, err error) {
	_, _ = conn.Write([]byte(fmt.Sprintf("-ERR %s\r\n", err.Error())))
}
