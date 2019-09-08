package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

	ctx, shutdown := context.WithCancel(context.Background())

	go WatcherOpenLibraries(self.LibsChan, time.Tick(10*time.Second), ctx)
	fsPathsToWatch := []string{
		usr.HomeDir,
		"/Volumes",
	}
	go WatcherPath(fsPathsToWatch, self.UpdateChan, ctx)

	go func(ctx context.Context) {
		ticker_6 := time.Tick(6 * time.Minute)
		ticker_15 := time.Tick(15 * time.Minute)
	mLoop:
		for {
			select {
			case <-ctx.Done():
				break mLoop
			case <-ticker_6:
				// Every 6 minutes
				self.Broadcast()
			case <-ticker_15:
				// Every 15 minutes
				self.SendAFK(LastUSBActivity())
			case <-self.Service.BroadcastChan:
				// Every time there are add/drops to the services list e.g. servers
				self.Broadcast()
			case libs := <-self.LibsChan:
				// Every time a library is opened/closed
				if !self.isSame(libs) {
					self.Libraries = libs
					self.Broadcast()
				}
			case lib := <-self.UpdateChan:
				// Every time something changes inside a library
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
		LogWarning("[STOP] Client services")
	}(ctx)

	err := self.Server(ctx, shutdown) // Blocking

	shutdown()
	time.Sleep(5 * time.Second) // Wait for everything to shutdown
	return err

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
			if IsConnectionRefused(err) {
				if !wasAWOL {
					self.AWOL[hostname] = time.Now()
					log.Printf("🏃💨 [%s] %s", hostname, url)
				} else if time.Since(lastSeen) > 24*time.Hour {
					delete(self.Service.Members, hostname)
					delete(self.AWOL, hostname)
					log.Printf("💔 [%s] %s", hostname, url)
				}
			} else {
				log.Println(err)
			}
			continue
		}
		b, _ := ioutil.ReadAll(res.Body)
		bs := string(b)
		if res.StatusCode != http.StatusOK {
			LogWarning(fmt.Sprintf("⚠️ [BROADCAST] [%s] %d: %s", hostname, res.StatusCode, bs))
			continue
		}
		if wasAWOL {
			delete(self.AWOL, hostname)
		}
		log.Println(fmt.Sprintf("[OK] [%s] %s", url, bs))
	}
}

func (self *Client) Post(route string, body []byte) {
	var res *http.Response
	var err error
	for hostname, url := range self.Service.Members {
		if !self.AWOL[hostname].IsZero() {
			continue
		}
		c := NewHTTPTimeoutClient()
		url := fmt.Sprintf("%s/%s", url, route)
		if body != nil {
			res, err = c.Post(url, "application/json", bytes.NewReader(body))
		} else {
			res, err = c.Head(url)
		}
		if err != nil {
			LogError(fmt.Sprintf("[%s] %s", route, err.Error()))
		}
		if res.StatusCode != http.StatusOK {
			b, _ := ioutil.ReadAll(res.Body)
			LogWarning(fmt.Sprintf("⚠️ [%s] [%s] %d: %s", route, hostname, res.StatusCode, string(b)))
		}
	}
}

func (self *Client) SendAFK(afk int64) {
	route := fmt.Sprintf("_afk/%s/%s", self.Hostname, strconv.FormatInt(afk, 10))
	self.Post(route, nil)
}

func (self *Client) Update(update ProjectUpdate) {
	body, _ := json.Marshal(update)
	self.Post("_update", body)
}

func (self *Client) toJSON() string {
	b, err := json.Marshal(self)
	if err != nil {
		return ""
	}
	return string(b)
}

func (self *Client) Server(ctx context.Context, shutdown context.CancelFunc) (err error) {

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

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Request-Method", "GET")
	})

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": self.Hostname})
	})

	r.POST("/alert", func(c *gin.Context) {
		b, _ := ioutil.ReadAll(c.Request.Body)
		m := NotifyMessage{}
		json.Unmarshal(b, &m)
		go exec.Command(alerter, m.Message).Run()
		c.JSON(200, gin.H{"alert": m.Message})
	})

	r.POST("/notify", func(c *gin.Context) {
		b, _ := ioutil.ReadAll(c.Request.Body)
		m := NotifyMessage{}
		json.Unmarshal(b, &m)
		go exec.Command(notifier,
			"-title", "TTWP Notification",
			"-sender", "com.ttwp.FCPXMonitor",
			"-actions", "OK",
			"-group", "1234",
			"-message", m.Message,
		).Run()
		c.JSON(200, gin.H{"notify": m.Message})
	})

	r.GET("/_ping", func(c *gin.Context) {
		c.JSON(200, gin.H{self.Hostname: "pong"})
	})

	r.GET("/_shutdown", func(c *gin.Context) {
		shutdown()
		c.JSON(200, gin.H{"ok": "shutdown"})
	})

	log.Printf("😎 client started: %s :%d [%s]", self.Hostname, self.Port, self.Service.TXTRecord["version"])

	go r.Run(fmt.Sprintf(":%d", self.Port))

	<-ctx.Done()

	return nil

}
