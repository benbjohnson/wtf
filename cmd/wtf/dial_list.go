package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http"
)

// DialListCommand represents a command for listing dials.
// This command provides a short output of just the name or a verbose output
// which includes the id, name, & invite URL.
type DialListCommand struct {
	ConfigPath string
}

// Run executes the command.
func (c *DialListCommand) Run(ctx context.Context, args []string) error {
	// Build a flag set to retrieve the config path & verbose flag.
	fs := flag.NewFlagSet("wtf-dial-list", flag.ContinueOnError)
	verbose := fs.Bool("v", false, "verbose")
	attachConfigFlags(fs, &c.ConfigPath)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load the configuration.
	config, err := ReadConfigFile(c.ConfigPath)
	if err != nil {
		return err
	}

	// Authenticate user with API key.
	ctx = wtf.NewContextWithUser(ctx, &wtf.User{APIKey: config.APIKey})

	// Build dial service and fetch list of dials user is a member of.
	dialService := http.NewDialService(http.NewClient(config.URL))
	dials, _, err := dialService.FindDials(ctx, wtf.DialFilter{})
	if err != nil {
		return err
	}

	// Iterate over dials and print out information.
	for _, dial := range dials {
		// If we are not in verbose mode, just print the name.
		if !*verbose {
			fmt.Println(dial.Name)
			continue
		}

		// If we are in verbose mode, print a tab-delimited list of fields.
		fmt.Printf(
			"%d\t%s\t%s\n",
			dial.ID,
			dial.Name,
			config.URL+"/invite/"+dial.InviteCode,
		)
	}

	return nil
}

// usage prints command usage information to STDOUT.
func (c *DialListCommand) usage() {
	fmt.Println(`
List dials you are a member of.

Usage:

	wtf dial list

Arguments:

	-v
	    Enable verbose output.
`[1:])
}
