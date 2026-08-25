package server

import (
	"fmt"
	"strings"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) execHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) != 0 {
		return "", fmt.Errorf("EXEC does not accept any arguments")
	}

	if clientCtx.ClientState != StateTransaction {
		return "", fmt.Errorf("EXEC WITHOUT MULTI")
	}

	var results strings.Builder

	fmt.Fprintf(&results, "*%d\r\n", len(clientCtx.TxQueue))

	for _, changed := range s.WatchedKeys {
		if changed {
			clientCtx.TxQueue = nil
			clientCtx.ClientState = StateNormal
			clear(s.WatchedKeys)

			return "*-1\r\n", nil
		}
	}

	for _, c := range clientCtx.TxQueue {
		handler := c.Handler
		output, err := handler(clientCtx, c)
		if err != nil {
			results.WriteString(parser.ErrorOutputParser(err.Error()))
			continue
		}

		results.WriteString(output)
	}

	clientCtx.TxQueue = nil
	clientCtx.ClientState = StateNormal
	clear(s.WatchedKeys)

	return results.String(), nil
}
