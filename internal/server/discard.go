package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) discardHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) != 0 {
		return "", fmt.Errorf("DISCARD does not accept any arguments")
	}

	if clientCtx.ClientState != StateTransaction {
		return "", fmt.Errorf("DISCARD without MULTI")
	}

	clientCtx.ClientState = StateNormal
	clientCtx.TxQueue = nil
	clear(s.WatchedKeys)

	return parser.SimpleStringOutputParser("OK"), nil
}
