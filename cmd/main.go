package main

import (
	"fmt"
	"net"
	"os"

	"github.com/deltron-fr/redis-server/internal/server"
)

const port = "6379"

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		fmt.Printf("Failed to bind to port %s: %v\n", port, err)
		os.Exit(1)
	}

	s := server.NewServer()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go s.HandleConn(conn)
	}
}
