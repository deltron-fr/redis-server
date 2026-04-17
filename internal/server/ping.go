package server

import "github.com/deltron-fr/redis-server/internal/parser"

func (s *Server) pingHandler(cmd Command) (string, error) {
	return parser.SimpleStringOutputParser("PONG"), nil
}
