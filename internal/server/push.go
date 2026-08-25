package server

import (
	"fmt"
	"slices"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) rPushHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) < 2 {
		return "", fmt.Errorf("RPUSH command requires at least two arguments")
	}

	key := cmd.Args[0]
	var length int

	s.Mu.Lock()
	if value, exists := s.ListStore[key]; !exists {
		s.ListStore[key] = cmd.Args[1:]
		length = len(cmd.Args) - 1
	} else {
		s.ListStore[key] = append(value, cmd.Args[1:]...)
		length = len(s.ListStore[key])
	}
	s.Mu.Unlock()

	go func() {
		for {
			waiter := <-s.WaiterQueueList // block until someone is waiting
			if !waiter.Expired.Load() {
				waiter.Ch <- struct{}{}
				break
			}
		}
	}()

	return parser.IntegerOutputParser(length), nil
}

func (s *Server) lPushHandler(clientCtx *Client, cmd Command) (string, error) {
	if len(cmd.Args) < 2 {
		return "", fmt.Errorf("LPUSH command requires at least two arguments")
	}

	key := cmd.Args[0]
	slices.Reverse(cmd.Args[1:])

	s.Mu.Lock()
	defer s.Mu.Unlock()

	newValue := append(cmd.Args[1:], s.ListStore[key]...)
	s.ListStore[key] = newValue

	return parser.IntegerOutputParser(len(s.ListStore[key])), nil
}
