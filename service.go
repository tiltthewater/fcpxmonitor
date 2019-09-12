package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/grandcat/zeroconf"
)

const (
	SERVICE_CLIENT = "_fcpxmclient._tcp"
	SERVICE_SERVER = "_fcpxmserver._tcp"
)

type StringMap map[string]string

func NewService(hostname string, port int, serviceName string, _txtrecord *StringMap) Service {
	txtrecord := map[string]string{}
	if _txtrecord != nil {
		txtrecord = *_txtrecord
	}
	return Service{
		Hostname:      hostname,
		Port:          port,
		Name:          serviceName,
		TXTRecord:     txtrecord,
		Members:       StringMap{},
		BroadcastChan: make(chan StringMap, 100), // Have a buffer so we can test without having a consumer
		ExitChan:      make(chan bool),
	}
}

type Service struct {
	Hostname      string
	Port          int
	Name          string
	TXTRecord     map[string]string
	Members       map[string]string
	BroadcastChan chan StringMap
	ExitChan      chan bool
}

func (self *Service) Stop() {
	self.ExitChan <- true
}

func (self *Service) Start() {
	s := SERVICE_CLIENT
	if self.Name == SERVICE_CLIENT {
		s = SERVICE_SERVER
	}
	go self.discover(s)
	self.broadcast()
}

/*
type ServiceEntry struct {
	ServiceRecord
	HostName string   `json:"hostname"` // Host machine DNS name
	Port     int      `json:"port"`     // Service Port
	Text     []string `json:"text"`     // Service info served as a TXT record
	TTL      uint32   `json:"ttl"`      // TTL of the service record
	AddrIPv4 []net.IP `json:"-"`        // Host machine IPv4 œaddress
	AddrIPv6 []net.IP `json:"-"`        // Host machine IPv6 address
}
*/

func (self *Service) callback(entry *zeroconf.ServiceEntry) error {
	hostname := entry.ServiceRecord.Instance
	okURLs := []string{}
	for _, ip := range entry.AddrIPv4 {
		url := fmt.Sprintf("http://%s:%d", ip.String(), entry.Port)
		c := NewHTTPTimeoutClient()
		res, err := c.Head(fmt.Sprintf("%s/_ping", url))
		if err != nil {
			return err
		} else if res.StatusCode != 200 {
			return errors.New(fmt.Sprintf("Received status code: %d from '%s'", res.StatusCode, url))
		} else {
			okURLs = append(okURLs, url)
		}
	}
	if len(okURLs) == 0 {
		return errors.New("No valid ips found for host: " + hostname)
	}
	self.Members[hostname] = okURLs[0]
	if self.BroadcastChan != nil { // There might be cases when you don't care about this, so you can `nil` it out
		self.BroadcastChan <- self.Members
	}
	return nil
}

func (self *Service) broadcast() {

	txtRecord := []string{}
	if self.TXTRecord != nil {
		for k, v := range self.TXTRecord {
			rec := fmt.Sprintf("%s=%s", k, v)
			txtRecord = append(txtRecord, rec)
		}
	}

	service, err := zeroconf.Register(
		self.Hostname, // service instance name
		self.Name,     // service type and protocl
		"local.",      // service domain
		self.Port,     // service port
		txtRecord,     // service metadata
		nil,           // register on all network interfaces
	)
	if err != nil {
		LogError(err.Error())
	}
	defer service.Shutdown()
	<-self.ExitChan
}

func (self *Service) discover(serviceName string) {

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		LogError(err.Error())
	}

	// Channel to receive discovered service entries
	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			// log.Println("Found service:", entry.ServiceInstanceName(), entry.Text)
			err := self.callback(entry)
			if err != nil {
				LogError(fmt.Sprintf("[SERVICE] " + err.Error()))
			} else {
				log.Printf("👋 [%s] %s", entry.Service, entry.ServiceRecord.Instance)
			}
		}
	}(entries)

	ctx := context.Background()

	err = resolver.Browse(ctx, serviceName, "local.", entries)
	if err != nil {
		LogError("[SERVICE] Failed to browse: " + err.Error())
	}

	<-ctx.Done()
}
