package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/deltron-fr/redis-server/internal/parser"
)

// setHandler processes the SET command. It supports setting a key with a value,
// and optionally with an expiry time specified in seconds (EX) or milliseconds (PX).
func (s *Server) setHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 2 && len(cmd.Args) != 4 {
		return "", fmt.Errorf("SET command requires exactly two or four arguments")
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	switch len(cmd.Args) {
	case 2:
		s.Store[cmd.Args[0]] = ValueStore{Value: cmd.Args[1], Expiry: nil}
	case 4:
		expiryType := cmd.Args[2]
		expiryTime := cmd.Args[3]
		parsedExpiryTime, err := handleExpiry(expiryType, expiryTime)
		if err != nil {
			return "", fmt.Errorf("error handling expiry: %v", err)
		}
		s.Store[cmd.Args[0]] = ValueStore{Value: cmd.Args[1], Expiry: &parsedExpiryTime}
	}

	return parser.SimpleStringOutputParser("OK"), nil
}

func handleExpiry(expiryType, expiryTime string) (time.Time, error) {
	expiryType = strings.ToLower(expiryType)
	switch expiryType {
	case "ex":
		duration, err := time.ParseDuration(expiryTime + "s")
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid expiry time: %v", err)
		}

		return time.Now().Add(duration), nil
	case "px":
		duration, err := time.ParseDuration(expiryTime + "ms")
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid expiry time: %v", err)
		}

		return time.Now().Add(duration), nil
	}

	return time.Time{}, fmt.Errorf("invalid expiry type: %s", expiryType)
}
