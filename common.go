package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	CBTODO = func(string, []byte) {}
	NOBODY = []byte{}
)

type NotifyMessage struct {
	Message string `json:"message"`
}

type T struct {
	T time.Time `json:"t"`
}

type ClientPayload struct {
	Hostname  string       `json:"hostname"`
	Port      int          `json:"port"`
	Libraries FCPLibraries `json:"projects"`
}

type Host struct {
	Hostname      string
	Port          int
	Service       Service
	AWOL          map[string]time.Time
	ResponseTimes map[string]float64
	Ctx           context.Context
	Shutdown      context.CancelFunc
	Root          string
	Router        *gin.Engine
}

func (self *Host) Init() {

	gin.SetMode(gin.ReleaseMode)

	runDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	self.Root = filepath.Join(runDir, "dist_server")

	ctx, shutdown := context.WithCancel(context.Background())

	self.Ctx = ctx
	self.Shutdown = shutdown

	self.AWOL = make(map[string]time.Time)
	self.ResponseTimes = make(map[string]float64)

	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Request-Method", "GET")
	})

	r.HEAD("/_ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	r.GET("/_pong", func(c *gin.Context) {
		b, _ := json.Marshal(T{time.Now()})
		c.Data(200, "application/json", b)
	})

	r.GET("/_shutdown", func(c *gin.Context) {
		shutdown()
		c.JSON(200, gin.H{"ok": "shutdown"})
	})

	self.Router = r

}

func (self *Host) HandleError(err error, otherHost string) {
	lastSeen, wasAWOL := self.AWOL[otherHost]
	if !wasAWOL {
		self.AWOL[otherHost] = time.Now()
		log.Printf("🏃💨 [%s]", otherHost)
	} else if time.Since(lastSeen) > 60*time.Minute {
		self.Remove(otherHost)
		log.Printf("💔 [%s]", otherHost)
	}
}

func (self *Host) Remove(hostname string) {
	delete(self.Service.Members, hostname)
	delete(self.AWOL, hostname)
	delete(self.ResponseTimes, hostname)
}

func (self *Host) Broadcast(route string, body []byte, callback func(string, []byte)) {

	/* Do NOT pass nil into this function, use `callback_todo` or `NOBODY` instead */

	var res *http.Response
	var err error

	for hostname, url := range self.Service.Members {

		c := NewHTTPTimeoutClient()
		url := fmt.Sprintf("%s/%s", url, route)

		if len(body) > 0 {
			res, err = c.Post(url, "application/json", bytes.NewReader(body))
		} else {
			res, err = c.Get(url)
		}

		if err != nil {
			self.HandleError(err, hostname)
			continue
		}

		delete(self.AWOL, hostname)
		defer res.Body.Close()
		b, _ := ioutil.ReadAll(res.Body)
		msg := fmt.Sprintf("[%s][%d][%s] %s", hostname, res.StatusCode, route, string(b))
		if res.StatusCode != http.StatusOK {
			msg = "⚠️ " + msg
		} else if callback != nil {
			go callback(hostname, b)
		}

	}

}

func Lag(t time.Time) float64 {
	lagNano := time.Now().Sub(t).Nanoseconds()
	return float64(lagNano) / 1000000.0
}

func (self *Host) CheckMembersAlive() {
	self.Broadcast("_pong", NOBODY, func(hostname string, body []byte) {
		t := T{}
		json.Unmarshal(body, &t)
		self.ResponseTimes[hostname] = Lag(t.T)
		log.Printf("⏱️ [%s] %.2f ms\n", hostname, self.ResponseTimes[hostname])
	})
}

func ClientFromJSON(b []byte) (ClientPayload, error) {
	c := ClientPayload{}
	err := json.Unmarshal(b, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}
