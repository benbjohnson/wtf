package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http"
)

// DialCreateCommand is a command for creating dials.
type DialCreateCommand struct {
	ConfigPath string
}

// Run executes the command.
func (c *DialCreateCommand) Run(ctx context.Context, args []string) error {
	// Create a flag set with parameters for the dial fields.
	fs := flag.NewFlagSet("wtf-dial-create", flag.ContinueOnError)
	name := fs.String("name", "", "dial name")
	attachConfigFlags(fs, &c.ConfigPath)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load the configuration.
	config, err := ReadConfigFile(c.ConfigPath)
	if err != nil {
		return err
	}

	// Authenticate the user with the API key from the config.
	ctx = wtf.NewContextWithUser(ctx, &wtf.User{APIKey: config.APIKey})

	// Build dial from arguments and issue creation request over HTTP.
	dial := &wtf.Dial{Name: *name}
	svc := http.NewDialService(http.NewClient(config.URL))
	if err := svc.CreateDial(ctx, dial); err != nil {
		return err
	}

	// Notify user of their new dial.
	fmt.Printf("Your %q dial has been created!\n\n", dial.Name)
	fmt.Printf("Please share this URL to invite others to contribute:\n\n")
	fmt.Printf("%s\n\n", config.URL+"/invite/"+dial.InviteCode)

	return nil
}

// usage print usage information for the command to STDOUT.
func (c *DialCreateCommand) usage() {
	fmt.Println(`
Create a new dial.

Usage:

	wtf dial create -name "My Dial"

Arguments:

	-name NAME
	    The name of the dial you are creating. Required.
`[1:])
}
