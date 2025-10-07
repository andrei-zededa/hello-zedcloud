package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/andrei-zededa/zededa-cloud-speedrun/hello-zedcloud/pkg/server"
)

//go:embed version
var version string // version is the version string of this web server app.

const quickIDNotRandom = "000000"

// quickID generate a small string random ID. Although unlikely the call to
// crypto/rand can fail and it such a case quickID just returns the fixed and
// obviously not random string of "000000". Since this is only used for logging
// in an *example* app ...
func quickID(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		// NOTE: Indeed returning this fixed string is not very useful.
		return quickIDNotRandom
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// getEnvOrDefault returns the value of the environment variable if it exists,
// (including if it exists but set to an empty value), otherwise returns the
// default value.
func getEnvOrDefault(envVar, def string) string {
	if v, ok := os.LookupEnv(envVar); ok {
		return v
	}

	return def
}

func main() {
	// Get default values from environment variables (CLI flags will take
	// precedence).
	listenDef := getEnvOrDefault("HELLO_LISTEN", ":8080")
	staticDef := getEnvOrDefault("HELLO_STATIC", "./static")
	bwLimitDef := getEnvOrDefault("HELLO_BW_LIMIT", "2GB")
	usernameDef := getEnvOrDefault("HELLO_USERNAME", "$RANDOM")
	passwordDef := getEnvOrDefault("HELLO_PASSWORD", "$RANDOM")

	// Define the CLI flags for the server.
	listen := flag.String("listen", listenDef, "The address (`host:port`) on which the server should listen to."+
		" Can also be set via the HELLO_LISTEN environment variable.")
	staticDir := flag.String("static", staticDef, "The directory from which to serve static files."+
		" Can also be set via the HELLO_STATIC environment variable.")
	bwLimitStr := flag.String("bw-limit", bwLimitDef, "Limit the read and write bandwidth (each, not combined) of the entire server."+
		" A string like `2m, 2mb, 2M or 2MB all meaning 2 megabytes per second`."+
		" Can also be set via the HELLO_BW_LIMIT environment variable.")
	userFlag := flag.String("username", usernameDef, "Username for HTTP basic authentication."+
		" Default: $RANDOM, meaning that a random username is generated."+
		" Set to an empty string to disable authentication."+
		" Can also be set via the HELLO_USERNAME environment variable.")
	passFlag := flag.String("password", passwordDef, "Password for HTTP basic authentication."+
		" Default: $RANDOM, meaning that a random password is generated."+
		" Can also be set via the HELLO_PASSWORD environment variable.")
	flag.Parse()

	// Handle $RANDOM for username and password.
	username := *userFlag
	password := *passFlag

	if len(username) > 0 && strings.EqualFold(username, "$RANDOM") {
		username = quickID(12)
		if username == quickIDNotRandom {
			fmt.Fprintf(os.Stderr, "Failed to generate a random username\n")
			os.Exit(1)
		}
	}

	if len(username) > 0 && strings.EqualFold(password, "$RANDOM") {
		password = quickID(24)
		if password == quickIDNotRandom {
			fmt.Fprintf(os.Stderr, "Failed to generate a random password\n")
			os.Exit(1)
		}
	}

	// Create server configuration.
	config := server.Config{
		Listen:    *listen,
		StaticDir: *staticDir,
		BwLimit:   *bwLimitStr,
		Username:  username,
		Password:  password,
		Version:   version,
	}

	// Create and start the server.
	srv, err := server.New(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Serve(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
