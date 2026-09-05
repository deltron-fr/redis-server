package server

import (
	"fmt"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) replConfHandler(clientCtx *Client, cmd Command) (string, error) {
	return parser.SimpleStringOutputParser("OK"), nil
}

func (s *Server) psyncHandler(clientCtx *Client, cmd Command) (string, error) {
	if s.ReplicaInfo.Role != Master {
		return "", fmt.Errorf("replica node can't perform synchronization")
	}

	out := fmt.Sprintf("FULLRESYNC %s %d", s.ReplicaInfo.MasterReplID, s.ReplicaInfo.MasterReplOffset)

	return parser.SimpleStringOutputParser(out), nil
}
