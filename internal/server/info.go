package server

import (
	"fmt"
	"strings"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func (s *Server) infoHandler(clientCtx *Client, cmd Command) (string, error) {
	var result strings.Builder

	fmt.Fprintf(&result, "role:%s\r\n", s.ReplicaInfo.Role)

	if s.ReplicaInfo.Role == Master {
		fmt.Fprintf(&result, "master_replid:%s\r\n", s.ReplicaInfo.MasterReplID)
		fmt.Fprintf(&result, "master_repl_offset:%d\r\n", s.ReplicaInfo.MasterReplOffset)
	}
	return parser.BulkStringOutputParser(result.String()), nil
}
