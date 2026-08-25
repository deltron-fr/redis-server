package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) multiHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) != 0 {
		return "", fmt.Errorf("MULTI does not accept any arguments")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	if clientCtx.ClientState == StateTransaction {
		return "", fmt.Errorf("MULTI calls can not be nested")
	}

	clientCtx.ClientState = StateTransaction

	return parser.BulkStringOutputParser("OK"), nil
}
