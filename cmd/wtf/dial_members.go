package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http"
)

// DialMembersCommand represents a command for listing members of a dial.
type DialMembersCommand struct {
	ConfigPath string
}

// Run executes the command.
func (c *DialMembersCommand) Run(ctx context.Context, args []string) error {
	// Create a flag set to read the config path & read the dial ID.
	fs := flag.NewFlagSet("wtf-dial-members", flag.ContinueOnError)
	attachConfigFlags(fs, &c.ConfigPath)
	if err := fs.Parse(args); err != nil {
		return err
	} else if fs.NArg() == 0 {
		return fmt.Errorf("Dial ID required.")
	} else if fs.NArg() > 1 {
		return fmt.Errorf("Only one dial ID allowed.")
	}

	// Parse dial ID from first arg.
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("Invalid dial ID.")
	}

	// Load configuration file.
	config, err := ReadConfigFile(c.ConfigPath)
	if err != nil {
		return err
	}

	// Authenticate user with API key.
	ctx = wtf.NewContextWithUser(ctx, &wtf.User{APIKey: config.APIKey})

	// Instantiate HTTP user service and fetch dial.
	// Members are automatically attached to the dial.
	dialService := http.NewDialService(http.NewClient(config.URL))
	dial, err := dialService.FindDialByID(ctx, id)
	if err != nil {
		return err
	}

	// Iterate over membrships and print the name & value.
	for _, membership := range dial.Memberships {
		fmt.Printf(
			"%s\t%d\n",
			membership.User.Name,
			membership.Value,
		)
	}

	return nil
}

// usage prints command usage information to STDOUT.
func (c *DialMembersCommand) usage() {
	fmt.Println(`
List members of a dial and their WTF level.

Usage:

	wtf dial members DIAL_ID
`[1:])
}
