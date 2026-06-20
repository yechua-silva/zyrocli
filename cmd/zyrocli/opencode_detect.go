package main

import (
	"net"
	"os"
	"time"
)

// IsInsideOpenCode checks whether the CLI is running inside an OpenCode session.
// It uses two methods:
//  1. OPENCODE_SESSION_ID environment variable
//  2. TCP probe to localhost:4096 (OpenCode SDK server)
func IsInsideOpenCode() bool {
	// Method 1: Environment variable
	if os.Getenv("OPENCODE_SESSION_ID") != "" {
		return true
	}

	// Method 2: Probe the OpenCode SDK server
	conn, err := net.DialTimeout("tcp", "localhost:4096", 100*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	return false
}
