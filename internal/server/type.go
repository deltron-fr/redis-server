package server

import (
	"fmt"
	"time"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) typeHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 1 {
		return "", fmt.Errorf("TYPE command requires exactly two arguments")
	}

	s.Mu.RLock()
	valueStore, okStore := s.Store[cmd.Args[0]]

	_, okStream := s.StreamStore[cmd.Args[0]]
	s.Mu.RUnlock()

	if !okStore && !okStream {
		return parser.SimpleStringOutputParser("none"), nil
	}

	if okStore {
		if valueStore.Expiry != nil && time.Now().After(*valueStore.Expiry) {
			return parser.SimpleStringOutputParser("none"), nil
		}
		return parser.SimpleStringOutputParser("string"), nil
	}

	if okStream {
		return parser.SimpleStringOutputParser("stream"), nil
	}

	return "", fmt.Errorf("unknown type")
}
