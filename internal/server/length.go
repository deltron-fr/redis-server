package server

import (
	"fmt"
)

func (s *Server) lLenHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 1 {
		return "", fmt.Errorf("LLEN command requires exactly one argument")
	}

	s.Mu.RLock()
	defer s.Mu.RUnlock()

	if len(s.ListStore[cmd.Args[0]]) == 0 {
		return ":0\r\n", nil // Return 0 if key does not exist or list is empty
	}

	return fmt.Sprintf(":%d\r\n", len(s.ListStore[cmd.Args[0]])), nil
}
