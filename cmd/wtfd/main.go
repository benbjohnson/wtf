package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/benbjohnson/wtf"
	"github.com/benbjohnson/wtf/http"
	"github.com/benbjohnson/wtf/inmem"
	"github.com/benbjohnson/wtf/sqlite"
	"github.com/pelletier/go-toml"
)

// Build version, injected during build.
var (
	version string
	commit  string
)

// main is the entry point to our application binary. However, it has some poor
// usability so we mainly use it to delegate out to our Main type.
func main() {
	// Propagate build information to root package to share globally.
	wtf.Version, wtf.Commit = version, commit

	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	// Instantiate a new type to represent our application.
	// This type lets us shared setup code with our end-to-end tests.
	m := NewMain()

	// Parse command line flags & load configuration.
	if err := m.ParseFlags(ctx, os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Execute program.
	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for CTRL-C.
	<-ctx.Done()

	// Clean up program.
	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the program.
type Main struct {
	// Configuration path and parsed config data.
	Config     Config
	ConfigPath string

	// SQLite database used by SQLite service implementations.
	DB *sqlite.DB

	// HTTP server for handling HTTP communication.
	// SQLite services are attached to it before running.
	HTTPServer *http.Server

	// Services exposed for end-to-end tests.
	UserService wtf.UserService
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Config:     DefaultConfig(),
		ConfigPath: DefaultConfigPath,

		DB:         sqlite.NewDB(""),
		HTTPServer: http.NewServer(),
	}
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}
	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
	}
	return nil
}

// ParseFlags parses the command line arguments & loads the config.
//
// This exists separately from the Run() function so that we can skip it
// during end-to-end tests. Those tests will configure manually and call Run().
func (m *Main) ParseFlags(ctx context.Context, args []string) error {
	// Our flag set is very simple. It only includes a config path.
	fs := flag.NewFlagSet("wtfd", flag.ContinueOnError)
	fs.StringVar(&m.ConfigPath, "config", DefaultConfigPath, "config path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// The expand() function is here to automatically expand "~" to the user's
	// home directory. This is a common task as configuration files are typing
	// under the home directory during local development.
	configPath, err := expand(m.ConfigPath)
	if err != nil {
		return err
	}

	// Read our TOML formatted configuration file.
	config, err := ReadConfigFile(configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", m.ConfigPath)
	} else if err != nil {
		return err
	}
	m.Config = config

	return nil
}

// Run executes the program. The configuration should already be set up before
// calling this function.
func (m *Main) Run(ctx context.Context) (err error) {
	// Initialize event service for real-time events.
	// We are using an in-memory implementation but this could be changed to
	// a more robust service if we expanded out to multiple nodes.
	eventService := inmem.NewEventService()

	// Attach our event service to the SQLite database so it can publish events.
	m.DB.EventService = eventService

	// Expand the DSN (in case it is in the user home directory ("~")).
	// Then open the database. This will instantiate the SQLite connection
	// and execute any pending migration files.
	if m.DB.DSN, err = expandDSN(m.Config.DB.DSN); err != nil {
		return fmt.Errorf("cannot expand dsn: %w", err)
	}
	if err := m.DB.Open(); err != nil {
		return fmt.Errorf("cannot open db: %w", err)
	}

	// Instantiate SQLite-backed services.
	authService := sqlite.NewAuthService(m.DB)
	dialService := sqlite.NewDialService(m.DB)
	dialMembershipService := sqlite.NewDialMembershipService(m.DB)
	userService := sqlite.NewUserService(m.DB)

	// Attach user service to Main for testing.
	m.UserService = userService

	// Copy configuration settings to the HTTP server.
	m.HTTPServer.Addr = m.Config.HTTP.Addr
	m.HTTPServer.Domain = m.Config.HTTP.Domain
	m.HTTPServer.HashKey = m.Config.HTTP.HashKey
	m.HTTPServer.BlockKey = m.Config.HTTP.BlockKey
	m.HTTPServer.GitHubClientID = m.Config.GitHub.ClientID
	m.HTTPServer.GitHubClientSecret = m.Config.GitHub.ClientSecret

	// Attach underlying services to the HTTP server.
	m.HTTPServer.AuthService = authService
	m.HTTPServer.DialService = dialService
	m.HTTPServer.DialMembershipService = dialMembershipService
	m.HTTPServer.EventService = eventService
	m.HTTPServer.UserService = userService

	// Start the HTTP server.
	if err := m.HTTPServer.Open(); err != nil {
		return err
	}

	// If TLS enabled, redirect non-TLS connections to TLS.
	if m.HTTPServer.UseTLS() {
		go http.ListenAndServeTLSRedirect()
	}

	log.Printf("running: url=%q dsn=%q", m.HTTPServer.URL(), m.Config.DB.DSN)

	return nil
}

const (
	// DefaultConfigPath is the default path to the application configuration.
	DefaultConfigPath = "~/wtfd.conf"

	// DefaultDSN is the default datasource name.
	DefaultDSN = "~/.wtfd/db"
)

// Config represents the CLI configuration file.
type Config struct {
	DB struct {
		DSN string `toml:"dsn"`
	} `toml:"db"`

	HTTP struct {
		Addr     string `toml:"addr"`
		Domain   string `toml:"domain"`
		HashKey  string `toml:"hash-key"`
		BlockKey string `toml:"block-key"`
	} `toml:"http"`

	GitHub struct {
		ClientID     string `toml:"client-id"`
		ClientSecret string `toml:"client-secret"`
	} `toml:"github"`
}

// DefaultConfig returns a new instance of Config with defaults set.
func DefaultConfig() Config {
	var config Config
	config.DB.DSN = DefaultDSN
	return config
}

// ReadConfigFile unmarshals config from
func ReadConfigFile(filename string) (Config, error) {
	config := DefaultConfig()
	if buf, err := ioutil.ReadFile(filename); err != nil {
		return config, err
	} else if err := toml.Unmarshal(buf, &config); err != nil {
		return config, err
	}
	return config, nil
}

// expand returns path using tilde expansion. This means that a file path that
// begins with the "~" will be expanded to prefix the user's home directory.
func expand(path string) (string, error) {
	// Ignore if path has no leading tilde.
	if path != "~" && !strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path, nil
	}

	// Fetch the current user to determine the home path.
	u, err := user.Current()
	if err != nil {
		return path, err
	} else if u.HomeDir == "" {
		return path, fmt.Errorf("home directory unset")
	}

	if path == "~" {
		return u.HomeDir, nil
	}
	return filepath.Join(u.HomeDir, strings.TrimPrefix(path, "~"+string(os.PathSeparator))), nil
}

// expandDSN expands a datasource name. Ignores in-memory databases.
func expandDSN(dsn string) (string, error) {
	if dsn == ":memory:" {
		return dsn, nil
	}
	return expand(dsn)
}
