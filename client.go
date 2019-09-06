package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/user"
	"time"

	"github.com/gin-gonic/gin"
)

type FCPLibraries map[string]*FCPLibrary

type NotifyMessage struct {
	Message string `json:"message"`
}

func NewClient(service Service) Client {
	return Client{
		Hostname:   service.Hostname,
		Port:       service.Port,
		Libraries:  FCPLibraries{},
		Service:    service,
		AWOL:       map[string]time.Time{},
		LibsChan:   make(chan FCPLibraries),
		UpdateChan: make(chan FCPLibrary),
	}
}

type Client struct {
	Hostname   string               `json:"hostname"`
	Port       int                  `json:"port"`
	Libraries  FCPLibraries         `json:"projects"`
	Service    Service              `json:"-"`
	AWOL       map[string]time.Time `json:"-"`
	LibsChan   chan FCPLibraries    `json:"-"`
	UpdateChan chan FCPLibrary      `json:"-"`
}

func (self *Client) Start() error {

	usr, _ := user.Current()
	ctx, cancel := context.WithCancel(context.Background())

	go WatcherOpenLibraries(self.LibsChan, time.Tick(10*time.Second), ctx)
	fsPathsToWatch := []string{
		usr.HomeDir,
		"/Volumes",
	}
	go WatcherPath(fsPathsToWatch, self.UpdateChan, ctx)
	defer cancel()

	log.Printf("[INFO] Client started: %s (%d)", self.Hostname, self.Port)

	go func() {
		for {
			select {
			case <-self.Service.BroadcastChan:
				/* Force broadbast whenever the members of the service
				changes e.g. add/drops of clients/servers
				*/
				self.Broadcast()
			case libs := <-self.LibsChan:
				if !self.isSame(libs) {
					self.Libraries = libs
					self.Broadcast()
				}
			case lib := <-self.UpdateChan:
				_, hasKey := self.Libraries[lib.UUID]
				if hasKey {
					self.Update(ProjectUpdate{
						Hostname: self.Hostname,
						UUID:     lib.UUID,
						Last:     lib.Last,
					})
				}
			}
		}
	}()
	return self.Server()
}

func (self *Client) isSame(libs FCPLibraries) bool {
	if len(self.Libraries) != len(libs) {
		return false
	}
	b1, _ := json.Marshal(self.Libraries)
	b2, _ := json.Marshal(libs)
	return string(b1) == string(b2)
}

func (self *Client) Broadcast() {
	cJSON := self.toJSON()
	for hostname, url := range self.Service.Members {
		lastSeen, wasAWOL := self.AWOL[hostname]
		c := NewHTTPTimeoutClient()
		route := fmt.Sprintf("%s/_checkout", url)
		res, err := c.Post(route, "application/json", bytes.NewBufferString(cJSON))
		if err != nil {
			if connection_refused(err) {
				if !wasAWOL {
					self.AWOL[hostname] = time.Now()
					log.Printf("[AWOL] [%s] %s", hostname, url)
				} else if time.Since(lastSeen) > 24*time.Hour {
					delete(self.Service.Members, hostname)
					delete(self.AWOL, hostname)
					log.Printf("[DROPPED] [%s] %s", hostname, url)
				}
			} else {
				log.Println(err)
			}
			continue
		}
		b, _ := ioutil.ReadAll(res.Body)
		bs := string(b)
		if res.StatusCode != http.StatusAccepted {
			log.Println(fmt.Sprintf("[WARNING] [%s] %d: %s", url, res.StatusCode, bs))
			continue
		}
		if wasAWOL {
			delete(self.AWOL, hostname)
		}
		log.Println(fmt.Sprintf("[OK] [%s] %s", url, bs))
	}
}

func (self *Client) Update(update ProjectUpdate) {
	for hostname, url := range self.Service.Members {
		if !self.AWOL[hostname].IsZero() {
			continue
		}
		c := NewHTTPTimeoutClient()
		route := fmt.Sprintf("%s/_update", url)
		b, _ := json.Marshal(update)
		res, err := c.Post(route, "application/json", bytes.NewReader(b))
		if err != nil {
			continue
		}
		if res.StatusCode != http.StatusOK {
			b, _ = ioutil.ReadAll(res.Body)
			bs := string(b)
			log.Println(fmt.Sprintf("[WARNING] [%s] %d: %s", url, res.StatusCode, bs))
		}
	}
}

func (self *Client) toJSON() string {
	b, err := json.Marshal(self)
	if err != nil {
		return ""
	}
	return string(b)
}

func (self *Client) Server() error {

	shutdown := make(chan bool)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Request-Method", "GET")
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": self.Hostname})
	})

	r.POST("/notify", func(c *gin.Context) {
		b, _ := ioutil.ReadAll(c.Request.Body)
		m := NotifyMessage{}
		json.Unmarshal(b, &m)
		go NotifyAlert("Alert", m.Message)
		c.JSON(200, gin.H{"notify": m.Message})
	})

	r.GET("/_ping", func(c *gin.Context) {
		c.JSON(200, gin.H{self.Hostname: "pong"})
	})

	r.GET("/_shutdown", func(c *gin.Context) {
		shutdown <- true
		c.JSON(200, gin.H{"ok": "shutdown"})
	})

	go r.Run(fmt.Sprintf(":%d", self.Port))

	<-shutdown

	return nil

}
