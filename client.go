package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	APP_BUNDLE = "/Applications/FCPXMonitor.app/Contents/Resources"
)

type FCPLibraries map[string]*FCPLibrary

func NewClient(service Service) Client {
	cl := Client{}
	cl.Init()
	cl.Hostname = service.Hostname
	cl.Port = service.Port
	cl.Service = service
	cl.Library = FCPLibraries{}
	cl.LibsChan = make(chan FCPLibraries)
	cl.UpdateChan = make(chan FCPLibrary)
	return cl
}

type Client struct {
	Host
	Library    FCPLibraries
	LibsChan   chan FCPLibraries
	UpdateChan chan FCPLibrary
}

func (self *Client) Start() error {

	usr, _ := user.Current()

	go WatcherOpenLibraries(self.LibsChan, time.Tick(10*time.Second), self.Ctx)
	fsPathsToWatch := []string{
		usr.HomeDir,
		"/Volumes",
	}
	go WatcherPath(fsPathsToWatch, self.UpdateChan, self.Ctx)

	go func(ctx context.Context) {
		ticker_6 := time.Tick(6 * time.Minute)
		ticker_15 := time.Tick(15 * time.Minute)
	mLoop:
		for {
			select {
			case <-ctx.Done():
				break mLoop
			case <-self.Service.BroadcastChan:
				// Every time there are add/drops to the services list e.g. servers
				self.ReportCheckouts()
			case libs := <-self.LibsChan:
				// Every time a library is opened/closed
				if !self.isSame(libs) {
					self.Library = libs
					self.ReportCheckouts()
				}
			case lib := <-self.UpdateChan:
				// Every time something changes inside a library
				_, hasKey := self.Library[lib.UUID]
				if hasKey {
					self.UpdateProjectActivity(ProjectUpdate{
						Hostname: self.Hostname,
						UUID:     lib.UUID,
						Last:     lib.Last,
					})
				}
			case <-ticker_15:
				// Every 15 minutes
				self.SendAFK(LastUSBActivity())
			case <-ticker_6:
				// Every 6 minutes
				self.ReportCheckouts()
			}
		}
		LogWarning("[STOP] Client services")
	}(self.Ctx)

	err := self.Listen() // Blocking

	self.Shutdown()
	time.Sleep(5 * time.Second) // Wait for everything to shutdown
	return err

}

func (self *Client) Listen() (err error) {

	alerter := filepath.Join(APP_BUNDLE, "alerter")
	_, err = os.Stat(alerter)
	if err != nil {
		return errors.New("[client.Server] alerter not found: " + alerter)
	}

	notifier := filepath.Join(APP_BUNDLE, "notifier")
	_, err = os.Stat(notifier)
	if err != nil {
		return errors.New("[client.Server] notifier not found: " + notifier)
	}

	r := self.Router

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": self.Hostname})
	})

	r.POST("/notify", func(c *gin.Context) {
		m := &NotifyMessage{}
		json.NewDecoder(c.Request.Body).Decode(m)
		noti := exec.Command(notifier,
			"-title", "TTWP Notification",
			"-sender", "com.ttwp.FCPXMonitor",
			"-actions", "OK",
			"-group", "1234",
			"-message", m.Message,
		)
		noti.Start()
		c.JSON(200, gin.H{"notify": m.Message})
	})

	// This doesn't really work. Alerts don't come to front.
	r.POST("/alert", func(c *gin.Context) {
		m := &NotifyMessage{}
		json.NewDecoder(c.Request.Body).Decode(m)
		go exec.Command(alerter, m.Message).Run()
		c.JSON(200, gin.H{"alert": m.Message})
	})

	log.Printf("😎 client started: %s :%d [%s]", self.Hostname, self.Port, self.Service.TXTRecord["version"])

	go r.Run(fmt.Sprintf(":%d", self.Port))

	<-self.Ctx.Done()

	return nil

}

func (self *Client) ReportCheckouts() {
	clientPayload := self.toJSON()
	self.Broadcast("_checkout", clientPayload, CBTODO)
}

func (self *Client) SendAFK(afk int64) {
	route := fmt.Sprintf("_afk/%s/%s", self.Hostname, strconv.FormatInt(afk, 10))
	self.Broadcast(route, NOBODY, CBTODO)
}

func (self *Client) UpdateProjectActivity(update ProjectUpdate) {
	body, _ := json.Marshal(update)
	self.Broadcast("_update", body, CBTODO)
}

func (self *Client) toJSON() []byte {
	b, err := json.Marshal(ClientPayload{
		Hostname:  self.Hostname,
		Port:      self.Port,
		Libraries: self.Library,
	})
	if err != nil {
		return []byte{}
	}
	return b
}

func (self *Client) isSame(libs FCPLibraries) bool {
	if len(self.Library) != len(libs) {
		return false
	}
	b1, _ := json.Marshal(self.Library)
	b2, _ := json.Marshal(libs)
	return string(b1) == string(b2)
}
