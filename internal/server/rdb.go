package server

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
)

func (s *Server) emptyRDBTransfer() {
	emptyRDBEncodedStr := "UkVESVMwMDEw+glyZWRpcy12ZXIGNy4wLjE1+gpyZWRpcy1iaXRzwED6BWN0aW1lwqh+mmr6CHVzZWQtbWVtwpBUDgD6CGFvZi1iYXNlwAD/oP/CldLqFoE="
	decBytes, err := base64.StdEncoding.DecodeString(emptyRDBEncodedStr)
	if err != nil {
		// TODO: do something
		return
	}

	fullBytes := fmt.Appendf([]byte{}, "$%d\r\n%s", len(decBytes), decBytes)
	r := bytes.NewReader(fullBytes)

	for client := range s.ResyncCh {
		r.Reset(fullBytes)
		bufferedConn := bufio.NewWriter(client.ClientConn)
		io.Copy(bufferedConn, r)
		bufferedConn.Flush()

		s.ReplicaInfo.Replicas[client] = true
	}
}
