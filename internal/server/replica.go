package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

func (s *Server) propagateToReplicas(input string) {
	for client := range s.ReplicaInfo.Replicas {
		_, err := io.Copy(client.ClientConn, strings.NewReader(input))
		if err != nil {
			fmt.Println(err)
		}
		if errors.Is(err, net.ErrClosed) {
			delete(s.ReplicaInfo.Replicas, client)
		}
	}
}
