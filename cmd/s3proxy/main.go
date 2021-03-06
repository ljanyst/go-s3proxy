//------------------------------------------------------------------------------
// Author: Lukasz Janyst <lukasz@jany.st>
// Date: 07.09.2019
//
// Licensed under the BSD License, see the LICENSE file for details.
//------------------------------------------------------------------------------

package main

import (
	"flag"
	"os"

	"github.com/ljanyst/go-s3proxy/pkg/s3proxy"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func main() {
	// Commandline
	configFile := flag.String("config", "", "a JSON configuration file")
	logFile := flag.String("log-file", "", "output file for diagnostics")
	logLevel := flag.String("log-level", "Info",
		"verbosity of the diagnostic information")
	flag.Parse()

	// Logging
	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		ForceFormatting: true,
	})

	log.Info("Starting s3proxy...")

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	level := log.InfoLevel
	if *logLevel != "" {
		l, err := log.ParseLevel(*logLevel)
		if err != nil {
			log.Fatal(err)
		}
		level = l
	}
	log.SetLevel(level)

	// Configuration
	opts := s3proxy.NewS3ProxyOpts()
	if *configFile != "" {
		err := opts.LoadYaml(*configFile)
		if err != nil {
			log.Fatalf("Failed to read configuration: %s", err)
		}
	}

	s3proxy.RunWebServer(opts)
}
