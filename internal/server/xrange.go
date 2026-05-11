package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) xRangeHandler(cmd Command) (string, error) {
	if len(cmd.Args) != 3 {
		return "", fmt.Errorf("invalid number of arguments")
	}

	key := cmd.Args[0]

	s.Mu.RLock()
	values, ok := s.StreamStore[key]
	s.Mu.RUnlock()

	fmt.Println("reached here?", ok)

	if !ok {
		return "*0\r\n", nil
	}

	start, stop := cmd.Args[1], cmd.Args[2]
	if start == "-" && len(values) >= 1 {
		start = values[0].ID
	}

	if stop == "+" && len(values) >= 2 {
		stop = values[len(values)-1].ID
	}

	var streamEntries []StreamEntry

	started := false
	for _, v := range values {
		if v.ID == start {
			streamEntries = append(streamEntries, v)
			started = true
			continue
		}

		if started && v.ID != stop {
			streamEntries = append(streamEntries, v)
			continue
		}

		if v.ID == stop {
			streamEntries = append(streamEntries, v)
			break
		}
	}

	if len(streamEntries) == 0 {
		return "*0/r/n", nil
	}

	var results []any
	for _, r := range streamEntries {
		var keyValueEntries []string
		for k, v := range r.Fields {
			keyValueEntries = append(keyValueEntries, k)
			keyValueEntries = append(keyValueEntries, v)
		}
		results = append(
			results,
			[]any{
				r.ID,
				keyValueEntries,
			})
	}

	return parser.RESPOutputParser(results), nil
}
