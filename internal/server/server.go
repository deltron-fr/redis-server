package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deltron-fr/redis-server/internal/parser"
)

type State int

const (
	StateNormal State = iota
	StateTransaction
)

type Server struct {
	Store             map[string]ValueStore
	ListStore         map[string][]string
	StreamStore       map[string][]StreamEntry
	Commands          map[string]CommandHandler
	Mu                sync.RWMutex
	WaiterQueueList   chan *Waiter
	WaiterQueueStream chan *Waiter
}

type ValueStore struct {
	Value  string
	Expiry *time.Time
}

type Waiter struct {
	Ch      chan struct{}
	Expired atomic.Bool
}

type StreamEntry struct {
	ID     string
	Fields map[string]string
}

func NewServer() *Server {
	waiterQ := make(chan *Waiter, 100)
	waiterQStream := make(chan *Waiter, 100)

	s := &Server{
		Store:             make(map[string]ValueStore),
		ListStore:         make(map[string][]string),
		StreamStore:       make(map[string][]StreamEntry),
		WaiterQueueList:   waiterQ,
		WaiterQueueStream: waiterQStream,
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
		"BLPOP":  s.bLPopHandler,
		"TYPE":   s.typeHandler,
		"XADD":   s.xaddHandler,
		"XRANGE": s.xRangeHandler,
		"XREAD":  s.xReadHandler,
		"INCR":   s.incrHandler,
		"MULTI":  s.multiHandler,
		"EXEC":   s.execHandler,
	}

	return s
}

// HandleConn reads RESP requests off the connection, dispatches each to the
// registered command handler, and writes the response back.
func (s *Server) HandleConn(conn net.Conn) {
	defer conn.Close()

	clientCtx := &Client{ClientState: StateNormal}

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

		if name == "EXEC" && clientCtx.ClientState == StateTransaction {
			resp, err := handler(clientCtx, Command{
				Handler: handler,
				Args:    args[1:],
			})
			if err != nil {
				writeErr(conn, err)
				continue
			}

			if _, err := conn.Write([]byte(resp)); err != nil {
				return
			}

			continue
		}

		if clientCtx.ClientState == StateTransaction {
			clientCtx.TxQueue = append(clientCtx.TxQueue, Command{
				Handler: handler,
				Args:    args[1:],
			})
			if _, err := conn.Write([]byte(parser.BulkStringOutputParser("QUEUED"))); err != nil {
				return
			}
			continue
		}

		resp, err := handler(clientCtx, Command{
			Handler: handler,
			Args:    args[1:],
		})
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
