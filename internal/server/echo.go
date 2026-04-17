package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

// echoHandler processes the ECHO command. It returns a bulk string if there's one argument, or an array if there are multiple arguments.
func (s *Server) echoHandler(cmd Command) (string, error) {
	switch len(cmd.Args) {
	case 0:
		return "", fmt.Errorf("ECHO command requires at least one argument")
	case 1:
		return parser.BulkStringOutputParser(cmd.Args[0]), nil
	default:
		return parser.ArrayOutputParser(cmd.Args), nil
	}
}
