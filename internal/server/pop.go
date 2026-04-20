package server

import (
	"fmt"
	"time"

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

func (s *Server) bLPopHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 2 {
		return "", fmt.Errorf("BLPOP command requires exactly three arguments")
	}

	key := cmd.Args[0]
	timerInput := cmd.Args[1]

	timer, err := time.ParseDuration(timerInput + "s")
	if err != nil {
		return "", fmt.Errorf("couldn't parse timeout value: %v", err)
	}

	if timer < 0 {
		return "", fmt.Errorf("BLPOP requires a positive integer")
	}

	s.Mu.Lock()
	for {
		value, exists := s.ListStore[key]
		if exists && len(value) > 0 {
			result := []string{key, value[0]}
			s.ListStore[key] = value[1:]
			s.Mu.Unlock()
			return parser.ArrayOutputParser(result), nil
		}

		s.Mu.Unlock()

		w := &Waiter{Ch: make(chan struct{})}
		s.WaiterQueue <- w

		if timer == 0 {
			<-w.Ch
		} else {
			select {
			case <-w.Ch:

			case <-time.After(timer):
				w.Expired.Store(true)
				return "*-1\r\n", nil
			}
		}

		s.Mu.Lock()
	}
}
