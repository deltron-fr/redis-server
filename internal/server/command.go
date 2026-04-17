package server

// Command carries the arguments passed to a handler (the command name itself is stripped before dispatch).
type Command struct {
	Args []string
}

// CommandHandler defines the function signature for command handlers.
type CommandHandler func(cmd Command) (string, error)
