package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"adrian.wtf/marvelname"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	SERVER = "server"
	CLIENT = "client"
)

var (
	re_mode = regexp.MustCompile(`^(server|client)$`)
	_mode   = kingpin.Arg("mode", "Run mode: server|client").Required().String()
)

func main() {

	// gin overrides log's format, here, we override it back
	log.SetFlags(3)

	kingpin.Parse()
	mode := *_mode

	if !re_mode.MatchString(mode) {
		LogFatal("Invalid mode. Options are 'server' or 'client'")
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = marvelname.Hostname()
	}

	hostname = strings.Replace(hostname, ".local", "", 1)

	port := map[string]int{
		SERVER: 14036,
		CLIENT: 12140,
	}

	serviceName := map[string]string{
		SERVER: SERVICE_SERVER,
		CLIENT: SERVICE_CLIENT,
	}

	// Get directory of this application
	runDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		LogFatal(err.Error())
	}

	updater, err := NewUpdater(runDir, port[mode])
	if err != nil {
		LogFatal(err.Error())
	}

	txtRecord := &StringMap{"version": updater.Hash[0:7]}
	service := NewService(hostname, port[mode], serviceName[mode], txtRecord)

	// Start autodiscovery service
	go service.Start()

	// Start the self updater
	go updater.Start(context.Background(), 60*time.Second)

	switch mode {
	case CLIENT:
		client := NewClient(service)
		err = client.Start() // Blocking main loop
		if err != nil {
			LogFatal(err.Error())
		}
	case SERVER:
		server := NewServer(service)
		err = server.Start() // Blocking main loop
		if err != nil {
			LogFatal(err.Error())
		}
	}

	service.Stop()

	log.Printf("🤕 %s shutdown: %s", mode, hostname)
	time.Sleep(5 * time.Second)

}

func noop(...interface{}) {

}
