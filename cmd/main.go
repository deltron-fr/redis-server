package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/deltron-fr/redis-server/internal/server"
)

func main() {
	portPtr := flag.Int("port", 6379, "servers configured port")
	replicaPtr := flag.String("replicaof", "", "replica details of server - `<MASTER_HOST> <MASTER_PORT>`")
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *portPtr))
	if err != nil {
		fmt.Printf("Failed to bind to port %d: %v\n", *portPtr, err)
		os.Exit(1)
	}

	parts := strings.Fields(*replicaPtr)
	if len(parts) != 2 && *replicaPtr != "" {
		fmt.Printf("Invalid input for replicaof")
		os.Exit(1)
	}

	s := server.NewServer(*replicaPtr)

	if s.ReplicaInfo.Role == server.Slave {
		if err = s.Handshake(*portPtr); err != nil {
			fmt.Printf("couldn't connect to master server: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("started server")
	for {
		fmt.Println("new connection accepted")
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go s.HandleConn(conn)
	}
}
