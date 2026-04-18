package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) lPopHandler(cmd Command) (string, error) {
	if len(cmd.Args) > 2 {
		return "", fmt.Errorf("LPOP command requires at most two arguments")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	listKey := cmd.Args[0]

	if len(s.ListStore[listKey]) == 0 {
		return "$-1\r\n", nil // Return nil bulk string if key does not exist or list is empty
	}

	if len(cmd.Args) == 1 {
		element := s.ListStore[listKey][0]
		s.ListStore[listKey] = s.ListStore[listKey][1:]

		return parser.BulkStringOutputParser(element), nil
	}

	parsedIdx, err := parseIndex(cmd.Args[1])
	if err != nil {
		return "", fmt.Errorf("invalid index: %v", err)
	}

	if parsedIdx > len(s.ListStore[listKey]) {
		parsedIdx = len(s.ListStore[listKey])
	}

	var elements []string
	for i := 0; i < parsedIdx; i++ {
		elements = append(elements, s.ListStore[listKey][i])
	}

	s.ListStore[listKey] = s.ListStore[listKey][parsedIdx:]

	return parser.ArrayOutputParser(elements), nil
}
