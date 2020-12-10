package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
)

// DialCommand represents a collection of dial-related subcommands.
type DialCommand struct{}

// Run executes the command which delegates to other subcommands.
func (c *DialCommand) Run(ctx context.Context, args []string) error {
	// Shift off the subcommand name, if available.
	var cmd string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}

	// Delegete to the appropriate subcommand.
	switch cmd {
	case "", "list":
		return (&DialListCommand{}).Run(ctx, args)
	case "create":
		return (&DialCreateCommand{}).Run(ctx, args)
	case "delete":
		return (&DialDeleteCommand{}).Run(ctx, args)
	case "members":
		return (&DialMembersCommand{}).Run(ctx, args)
	case "help":
		c.usage()
		return flag.ErrHelp
	default:
		return fmt.Errorf("wtf dial %s: unknown command", cmd)
	}
}

// usage prints the subcommand usage to STDOUT.
func (c *DialCommand) usage() {
	fmt.Println(`
Manage WTF dials you own or are a member of.

Usage:

	wtf dial <command> [arguments]

The commands are:

	list        list all available dials
	create      create a new dial
	delete      remove an existing dial
	members     view list of members of a dial
`[1:])
}
