package server

import "fmt"

func (s *Server) rPushHandler(cmd Command) (string, error) {
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

	return fmt.Sprintf(":%d\r\n", length), nil
}
