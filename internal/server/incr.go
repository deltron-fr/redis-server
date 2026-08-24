package server

import (
	"fmt"
	"strconv"
	"time"
)

func (s *Server) incrHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 1 {
		return "", fmt.Errorf("INCR requires a single argument")
	}

	key := cmd.Args[0]

	s.Mu.Lock()
	defer s.Mu.Unlock()

	value, ok := s.Store[key]
	if !ok {
		s.Store[key] = ValueStore{Value: "1", Expiry: nil}
		return fmt.Sprintf(":%d\r\n", 1), nil
	}

	if value.Expiry != nil && time.Now().After(*value.Expiry) {
		return "$-1\r\n", nil // nil bulk string if key has expired
	}

	parsedValue, err := strconv.Atoi(value.Value)
	if err != nil {
		return "", fmt.Errorf("value is not an integer or out of range")
	}

	parsedValue++
	s.Store[key] = ValueStore{Value: strconv.Itoa(parsedValue), Expiry: nil}
	return fmt.Sprintf(":%d\r\n", parsedValue), nil
}
