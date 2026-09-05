package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/deltron-fr/redis-server/internal/parser"
)

// HandleConn reads RESP requests off the connection, dispatches each to the
// registered command handler, and writes the response back.
func (s *Server) HandleConn(conn net.Conn) {
	bufferedConn := bufio.NewWriter(conn)
	defer conn.Close()

	clientCtx := &Client{ClientConn: conn, ClientState: StateNormal}

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Println("read error:", err)
			}
			return
		}

		data := buf[:n]

		args, err := parser.Parse(data)
		if err != nil {
			writeErr(conn, err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		name := strings.ToUpper(args[0])
		command, ok := s.Commands[name]
		if !ok {
			writeErr(conn, fmt.Errorf("unknown command %q", name))
			continue
		}

		if isTxCommand(name) && clientCtx.ClientState == StateTransaction {
			resp, err := command.Handler(clientCtx, Command{
				Handler: command.Handler,
				Args:    args[1:],
			})
			if err != nil {
				writeErr(conn, err)
				continue
			}

			if _, err := bufferedConn.Write([]byte(resp)); err != nil {
				return
			}

			continue
		}

		if clientCtx.ClientState == StateTransaction {
			clientCtx.TxQueue = append(clientCtx.TxQueue, Command{
				Handler: command.Handler,
				Args:    args[1:],
			})
			if _, err := bufferedConn.Write([]byte(parser.BulkStringOutputParser("QUEUED"))); err != nil {
				return
			}
			continue
		}

		resp, err := command.Handler(clientCtx, Command{
			Handler: command.Handler,
			Args:    args[1:],
		})
		if err != nil {
			writeErr(conn, err)
			continue
		}

		respReader := strings.NewReader(resp)
		_, err = io.Copy(bufferedConn, respReader)
		if err != nil {
			return
		}

		if err = bufferedConn.Flush(); err != nil {
			return
		}

		if command.Type == WriteCommand && s.ReplicaInfo.Role == Master {
			s.propagateToReplicas(string(data))
		}
		if name == "PSYNC" {
			s.ResyncCh <- clientCtx 
		}
	}
}

func (s *Server) Handshake(port int) error {
	address := net.JoinHostPort(s.ReplicaInfo.MasterHost, s.ReplicaInfo.MasterPort)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	if _, err = conn.Write([]byte(parser.ArrayOutputParser([]string{"PING"}))); err != nil {
		return err
	}

	buf := make([]byte, 32)

	n, err := conn.Read(buf)
	if err != nil {
		return err
	}

	if string(buf[:n]) != parser.SimpleStringOutputParser("PONG") {
		return fmt.Errorf("invalid response to PING command")
	}
	clear(buf)

	_, err = conn.Write([]byte(parser.ArrayOutputParser([]string{"REPLCONF", "listening-port", strconv.Itoa(port)})))
	if err != nil {
		return err
	}

	n, err = conn.Read(buf)
	if err != nil {
		return err
	}

	if string(buf[:n]) != parser.SimpleStringOutputParser("OK") {
		return fmt.Errorf("invalid response to replication config command")
	}
	clear(buf)

	_, err = conn.Write([]byte(parser.ArrayOutputParser([]string{"REPLCONF", "capa", "psync2"})))
	if err != nil {
		return err
	}

	n, err = conn.Read(buf)
	if err != nil {
		return err
	}

	if string(buf[:n]) != parser.SimpleStringOutputParser("OK") {
		return fmt.Errorf("invalid response to replication config command")
	}
	clear(buf)

	_, err = conn.Write([]byte(parser.ArrayOutputParser([]string{"PSYNC", "?", "-1"})))
	if err != nil {
		return err
	}

	return nil
}

func writeErr(conn net.Conn, err error) {
	_, _ = conn.Write([]byte(fmt.Sprintf("-ERR %s\r\n", err.Error())))
}

func isTxCommand(cmdName string) bool {
	return cmdName == "EXEC" || cmdName == "DISCARD" || cmdName == "WATCH" ||
		cmdName == "MULTI" || cmdName == "UNWATCH"
}
