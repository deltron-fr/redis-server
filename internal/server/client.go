package server

import "net"

type State int

const (
	StateNormal State = iota
	StateTransaction
)

type CommandType int

const (
	WriteCommand CommandType = iota
	ReadCommand
)

type Client struct {
	ClientConn  net.Conn
	ClientState State
	TxQueue     []Command
}

// Command carries the arguments passed to a handler (the command name itself is stripped before dispatch).
type Command struct {
	Handler CommandHandler
	Args    []string
	Type    CommandType
}

// CommandHandler defines the function signature for command handlers.
type CommandHandler func(clientCtx *Client, cmd Command) (string, error)
