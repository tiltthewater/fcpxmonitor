package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

func NewServer(service Service) Server {
	return Server{
		Hostname: service.Hostname,
		Port:     service.Port,
		Service:  service,
		Library:  NewLibrary(),
	}
}

type AFK struct {
	sync.Mutex
	Map map[string]int
}

type Server struct {
	Hostname string
	Port     int
	Service  Service
	Library  Library
	AFK      *AFK
}

type CheckoutResponse struct {
	Checkout []string `json:"checkouts"`
	Closed   []string `json:"closed"`
	Errors   []string `json:"errors"`
}

func NewCheckout() CheckoutResponse {
	return CheckoutResponse{
		Checkout: []string{},
		Closed:   []string{},
		Errors:   []string{},
	}
}

func (self *Server) Start() error {

	runDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	site_root := filepath.Join(runDir, "dist_server")

	ctx, shutdown := context.WithCancel(context.Background())

	afks := AFK{Map: map[string]int{}}

	go self.Monitor(ctx)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Request-Method", "GET")
	})

	r.GET("/", func(c *gin.Context) {
		// k := Kobako["index.html"]
		// c.Header("Content-Encoding", "gzip")
		// c.Data(200, k.contentType, k.data)
		c.File(filepath.Join(site_root, "index.html"))
	})

	r.GET("/js/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		// k := Kobako[filename]
		// c.Header("Content-Encoding", "gzip")
		// c.Data(200, k.contentType, k.data)
		c.File(filepath.Join(site_root, filename))
	})

	r.GET("/library", func(c *gin.Context) {
		c.JSON(200, self.Library)
	})

	r.GET("/members", func(c *gin.Context) {
		c.JSON(200, self.Service.Members)
	})

	r.GET("/afks", func(c *gin.Context) {
		c.JSON(200, afks.Map)
	})

	r.GET("/_ping", func(c *gin.Context) {
		c.JSON(200, gin.H{self.Hostname: "pong"})
	})

	r.HEAD("/_afk/:hostname/:secondsaway", func(c *gin.Context) {
		hostname := c.Param("hostname")
		secondsaway, _ := strconv.Atoi(c.Param("secondsaway"))
		afks.Lock()
		defer afks.Unlock()
		self.AFK.Map[hostname] = secondsaway
		c.String(200, "ok")
	})

	r.POST("/_checkout", self.POST_Checkout)

	r.POST("/_update", self.POST_Update)

	r.GET("/_shutdown", func(c *gin.Context) {
		shutdown()
		c.JSON(200, gin.H{"ok": "shutdown"})
	})

	r.GET("/_legs", func(c *gin.Context) {
		// k := Kobako["abby.jpg"]
		// c.Header("Content-Encoding", "gzip")
		// c.Data(200, k.contentType, k.data)
		c.File(filepath.Join(site_root, "abby.jpg"))
	})

	log.Printf("😎 server started: %s :%d [%s]", self.Hostname, self.Port, self.Service.TXTRecord["version"])

	go r.Run(fmt.Sprintf(":%d", self.Port))

	<-ctx.Done()

	return nil

}

func (self *Server) POST_Checkout(c *gin.Context) {
	b, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		LogError(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cl, err := ClientFromJSON(b)
	if err != nil {
		LogError(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	checkout := self.Checkout(cl)
	c.JSON(http.StatusOK, checkout)
}

func (self *Server) POST_Update(c *gin.Context) {
	b, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		LogError(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	update := ProjectUpdate{}
	json.Unmarshal(b, &update)

	self.Library.Lock()
	defer self.Library.Unlock()

	status, err := self.Library.Update(update)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": fmt.Sprintf("%d", update.Last)})
}

func (self *Server) Checkout(cl *Client) CheckoutResponse {

	self.Library.Lock()
	defer self.Library.Unlock()

	checkout := NewCheckout()

	uuids := map[string]bool{}
	for uuid, lib := range cl.Libraries {
		err := self.Library.CheckoutProject(uuid, lib.Name, cl.Hostname, lib.Path, lib.Info)
		if err != nil {
			checkout.Errors = append(checkout.Errors, uuid)
			LogError(fmt.Sprintf("[%s] %s", cl.Hostname, err.Error()))
		} else {
			checkout.Checkout = append(checkout.Checkout, uuid)
			log.Printf("✅ [%s] by %s", uuid, cl.Hostname)
		}
		uuids[uuid] = true
	}

	closed, removed := self.Library.DeregisterProjects(cl.Hostname, uuids)
	checkout.Closed = closed
	for stat, dd := range map[string][]string{"CLOSED": closed, "REMOVED": removed} {
		for _, uuid := range dd {
			log.Printf("⚠ [%s] %s", stat, uuid)
		}
	}

	return checkout

}

func (self *Server) Monitor(ctx context.Context) {
	for {
		// No use server-side for now
		<-self.Service.BroadcastChan
	}
}

func ClientFromJSON(b []byte) (*Client, error) {
	c := &Client{}
	err := json.Unmarshal(b, c)
	if err != nil {
		return c, err
	}
	return c, nil
}
