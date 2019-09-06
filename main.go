package main

import (
	"log"
	"os"
	"regexp"
	"strings"
	"time"

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

	log.SetFlags(3)

	kingpin.Parse()
	mode := *_mode

	if !re_mode.MatchString(mode) {
		log.Fatal("Invalid mode. Options are 'server' or 'client'")
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "nobody"
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

	service := NewService(hostname, port[mode], serviceName[mode])
	go service.Start()

	switch mode {
	case CLIENT:
		client := NewClient(service)
		client.Start()
	case SERVER:
		server := NewServer(service)
		server.Start()
	}

	service.Stop()

	log.Printf("[INFO] %s shutdown: %s", mode, hostname)
	time.Sleep(5 * time.Second)

}

func noop(...interface{}) {

}
