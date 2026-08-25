package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)


// TODO: current implementation returns nil even if a key for another stream exists
func (s *Server) xReadHandler(clientCtx *Client, cmd Command) (string, error) {
	parts := cmd.Args[1:]
	if len(parts)%2 != 0 {
		return "", fmt.Errorf("invalid number of arguments")
	}

	mid := len(parts) / 2

	keys := parts[:mid]
	ids := parts[mid:]

	var results []any

	for i, key := range keys {
		s.Mu.Lock()
		entries, ok := s.StreamStore[key]
		s.Mu.Unlock()

		if !ok {
			return "*-1\r\n", nil // nil array if key doesn't exist
		}

		id := ids[i]
		var idx int
		var found bool
		for i, e := range entries {
			if e.ID > id {
				found = true
				idx = i
				break
			}
		}

		if !found {
			return "*-1\r\n", nil
		}

		newEntries := make([]StreamEntry, len(entries)-idx)
		copy(newEntries, entries[idx:])

		var streamResults []any

		for _, r := range newEntries {
			var keyValueEntries []string
			for k, v := range r.Fields {
				keyValueEntries = append(keyValueEntries, k)
				keyValueEntries = append(keyValueEntries, v)
			}

			streamResults = append(
				streamResults,
				[]any{
					r.ID,
					keyValueEntries,
				})
		}

		results = append(results, []any{key, streamResults})
	}

	return parser.RESPOutputParser(results), nil
}
