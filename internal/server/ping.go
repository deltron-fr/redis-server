package server

import "github.com/deltron-fr/redis-server/internal/parser"

func (s *Server) pingHandler(clientCtx *Client, cmd Command) (string, error) {
	return parser.SimpleStringOutputParser("PONG"), nil
}
