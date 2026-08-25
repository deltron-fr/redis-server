package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) lLenHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) != 1 {
		return "", fmt.Errorf("LLEN command requires exactly one argument")
	}

	s.Mu.RLock()
	defer s.Mu.RUnlock()

	if len(s.ListStore[cmd.Args[0]]) == 0 {
		return parser.IntegerOutputParser(0), nil // Return 0 if key does not exist or list is empty
	}

	return parser.IntegerOutputParser(len(s.ListStore[cmd.Args[0]])), nil
}
