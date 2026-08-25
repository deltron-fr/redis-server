package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) watchHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) < 1 {
		return "", fmt.Errorf("WATCH requires atleast a single key")
	}

	if clientCtx.ClientState == StateTransaction {
		return "", fmt.Errorf("WATCH inside MULTI is not allowed")
	}

	keys := cmd.Args[:]

	for _, key := range keys {
		s.WatchedKeys[key] = false
	}

	return parser.SimpleStringOutputParser("OK"), nil
}

func (s *Server) unwatchHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) != 0 {
		return "", fmt.Errorf("UNWATCH does not accept any arguments")
	}

	clear(s.WatchedKeys)

	return parser.SimpleStringOutputParser("OK"), nil
}
