package server

import (
	"fmt"
	"time"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) getHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 1 {
		return "", fmt.Errorf("GET command requires exactly one argument")
	}

	s.Mu.RLock()
	value, exists := s.Store[cmd.Args[0]]
	s.Mu.RUnlock()

	if !exists {
		return "$-1\r\n", nil // nil bulk string if key does not exist
	}

	if value.Expiry != nil && time.Now().After(*value.Expiry) {
		return "$-1\r\n", nil // nil bulk string if key has expired
	}

	return parser.BulkStringOutputParser(value.Value), nil
}
