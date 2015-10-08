/*
cfssl is the command line tool to issue/sign/bundle client certificate. It's
also a tool to start a HTTP server to handle web requests for signing, bundling
and verification.

Usage:
	cfssl command [-flags] arguments

	The commands are

	bundle	 create a certificate bundle
	sign	 signs a certificate signing request (CSR)
	serve	 starts a HTTP server handling sign and bundle requests
	version	 prints the current cfssl version
	genkey   generates a key and an associated CSR
	gencert  generates a key and a signed certificate
	selfsign generates a self-signed certificate

Use "cfssl [command] -help" to find out more about a command.
*/
package main

import (
	"flag"
	"os"

	"github.com/bbandix/cfssl/cli"
	"github.com/bbandix/cfssl/cli/bundle"
	"github.com/bbandix/cfssl/cli/gencert"
	"github.com/bbandix/cfssl/cli/genkey"
	"github.com/bbandix/cfssl/cli/info"
	"github.com/bbandix/cfssl/cli/ocspserve"
	"github.com/bbandix/cfssl/cli/ocspsign"
	"github.com/bbandix/cfssl/cli/printdefault"
	"github.com/bbandix/cfssl/cli/scan"
	"github.com/bbandix/cfssl/cli/selfsign"
	"github.com/bbandix/cfssl/cli/serve"
	"github.com/bbandix/cfssl/cli/sign"
	"github.com/bbandix/cfssl/cli/version"
	"github.com/bbandix/cfssl/log"
)

// main defines the cfssl usage and registers all defined commands and flags.
func main() {
	// Add command names to cfssl usage
	flag.Usage = nil // this is set to nil for testabilty
	flag.IntVar(&log.Level, "loglevel", log.LevelInfo, "Log level")
	// Register commands.
	cmds := map[string]*cli.Command{
		"bundle":         bundle.Command,
		"sign":           sign.Command,
		"serve":          serve.Command,
		"version":        version.Command,
		"genkey":         genkey.Command,
		"gencert":        gencert.Command,
		"ocspsign":       ocspsign.Command,
		"ocspserve":      ocspserve.Command,
		"selfsign":       selfsign.Command,
		"scan":           scan.Command,
		"info":           info.Command,
		"print-defaults": printdefaults.Command,
	}

	// If the CLI returns an error, exit with an appropriate status
	// code.
	err := cli.Start(cmds)
	if err != nil {
		os.Exit(1)
	}
}
