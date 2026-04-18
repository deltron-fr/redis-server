package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) lRangeHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 3 {
		return "", fmt.Errorf("LRANGE command requires exactly three arguments")
	}

	key := cmd.Args[0]
	start, err := parseIndex(cmd.Args[1])
	if err != nil {
		return "", fmt.Errorf("invalid start index: %v", err)
	}

	stop, err := parseIndex(cmd.Args[2])
	if err != nil {
		return "", fmt.Errorf("invalid stop index: %v", err)
	}

	s.Mu.RLock()
	value, exists := s.ListStore[key]
	s.Mu.RUnlock()

	if !exists {
		return "*0\r\n", nil // Return empty array if key does not exist
	}

	if start >= len(value) {
		return "*0\r\n", nil // Return empty array if start index is out of range
	}

	if start < 0 {
		start = len(value) + start
		start = max(start, 0) // Adjust start index if it's still negative after adjustment
	}

	if stop < 0 {
		stop = len(value) + stop
		stop = max(stop, 0) // Adjust stop index if it's still negative after adjustment
	}

	if stop >= len(value) {
		stop = len(value) - 1 // Adjust stop index if it's out of range
	}

	if start > stop {
		return "*0\r\n", nil // Return empty array if start index is greater than stop index
	}

	return parser.ArrayOutputParser(value[start : stop+1]), nil
}
