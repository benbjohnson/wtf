package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/benbjohnson/wtf"
	"github.com/pelletier/go-toml"
)

// Default settings
const (
	DefaultConfigPath = "~/wtf.config"
	DefaultURL        = "https://wtfdial.com"
)

// main is the entry point into our application. However, it provides poor
// usability since it does not allow us to return errors like most Go programs.
// Instead, we delegate most of our program to the Run() function.
func main() {
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Execute program.
	//
	// If an ErrHelp error is returned then that means the user has used an "-h"
	// flag and the flag package will handle output. We just need exit.
	//
	// If we have an application error (wtf.Error) then we can just display the
	// message. If we have any other error, print the raw error message.
	var e *wtf.Error
	if err := Run(ctx, os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if errors.As(err, &e) {
		fmt.Fprintln(os.Stderr, e.Message)
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Run executes the main program.
func Run(ctx context.Context, args []string) error {
	// Shift off subcommand from the argument list, if available.
	var cmd string
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	// Delegate subcommands to their own Run() methods.
	switch cmd {
	case "dial":
		return (&DialCommand{}).Run(ctx, args)
	case "", "-h", "help":
		usage()
		return flag.ErrHelp
	default:
		return fmt.Errorf("wtf %s: unknown command", cmd)
	}
}

// usage prints the top-level CLI usage message.
func usage() {
	fmt.Println(`
Command line utility for interacting with the WTF Dial service.

Usage:

	wtf <command> [arguments]

The commands are:

	dial        manage your dial
`[1:])
}

// Config represents a configuration file common to all subcommands.
type Config struct {
	// Base URL of the server. This should be changed for local development.
	URL string `toml:"url"`

	// API key used for authentication. Users can find this key on the /settings page.
	APIKey string `toml:"api-key"`
}

// DefaultConfig returns a new instance of Config with defaults set.
func DefaultConfig() Config {
	return Config{
		URL: DefaultURL,
	}
}

// ReadConfigFile unmarshals config from filename. Expands path if needed.
func ReadConfigFile(filename string) (Config, error) {
	config := DefaultConfig()

	// Expand filename, if necessary. This means substituting a "~" prefix
	// with the user's home directory, if available.
	if prefix := "~" + string(os.PathSeparator); strings.HasPrefix(filename, prefix) {
		u, err := user.Current()
		if err != nil {
			return config, err
		} else if u.HomeDir == "" {
			return config, fmt.Errorf("home directory unset")
		}
		filename = filepath.Join(u.HomeDir, strings.TrimPrefix(filename, prefix))
	}

	// Read & deserialize configuration.
	if buf, err := ioutil.ReadFile(filename); os.IsNotExist(err) {
		return config, fmt.Errorf("config file not found: %s", filename)
	} else if err != nil {
		return config, err
	} else if err := toml.Unmarshal(buf, &config); err != nil {
		return config, err
	}
	return config, nil
}

// attachConfigFlags adds a common "-config" flag to a flag set.
func attachConfigFlags(fs *flag.FlagSet, p *string) {
	fs.StringVar(p, "config", DefaultConfigPath, "config path")
}
