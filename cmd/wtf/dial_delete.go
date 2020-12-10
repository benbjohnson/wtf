package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http"
)

// DialDeleteCommand represents a command for deleting dials.
type DialDeleteCommand struct {
	ConfigPath string
}

// Run executes the command.
func (c *DialDeleteCommand) Run(ctx context.Context, args []string) error {
	// Create flag set to parse the config path & read the ID.
	fs := flag.NewFlagSet("wtf-dial-delete", flag.ContinueOnError)
	attachConfigFlags(fs, &c.ConfigPath)
	if err := fs.Parse(args); err != nil {
		return err
	} else if fs.NArg() == 0 {
		return fmt.Errorf("Dial ID required.")
	} else if fs.NArg() > 1 {
		return fmt.Errorf("Only one dial dial ID allowed.")
	}

	// Parse the dial ID from the first arg.
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("Invalid dial ID.")
	}

	// Load configuration file.
	config, err := ReadConfigFile(c.ConfigPath)
	if err != nil {
		return err
	}

	// Authenticate user using the API key.
	ctx = wtf.NewContextWithUser(ctx, &wtf.User{APIKey: config.APIKey})

	// Instantiate HTTP service and issue delete.
	svc := http.NewDialService(http.NewClient(config.URL))
	if err := svc.DeleteDial(ctx, id); err != nil {
		return err
	}

	// Notify user that dial is gone.
	fmt.Printf("Your dial has been deleted.\n")

	return nil
}

// usage prints the command usage information to STDOUT.
func (c *DialDeleteCommand) usage() {
	fmt.Println(`
Delete an existing dial.

Usage:

	wtf dial delete DIAL_ID
`[1:])
}
