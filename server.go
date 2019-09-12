package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func NewServer(service Service) Server {
	cl := Server{}
	cl.Init()
	cl.Hostname = service.Hostname
	cl.Port = service.Port
	cl.Service = service
	cl.Library = NewLibrary()
	cl.BroadcastChan = make(chan NotifyMessage)
	return cl
}

type AFK struct {
	sync.Mutex
	Map map[string]int
}

type Server struct {
	Host
	Library       Library
	BroadcastChan chan NotifyMessage
}

type CheckoutResponse struct {
	Checkouts []string `json:"checkouts"`
	Closed    []string `json:"closed"`
	Errors    []string `json:"errors"`
}

func NewCheckout() CheckoutResponse {
	return CheckoutResponse{
		Checkouts: []string{},
		Closed:    []string{},
		Errors:    []string{},
	}
}

func (self *Server) Start() error {

	go func(ctx context.Context) {
		ticker_5 := time.Tick(5 * time.Minute)
	mLoop:
		for {
			select {
			case <-ctx.Done():
				break mLoop
			case m := <-self.BroadcastChan:
				b, _ := json.Marshal(m)
				self.Broadcast("notify", b, CBTODO)
			case <-ticker_5:
				self.CheckMembersAlive()
			case <-self.Service.BroadcastChan:
				// No use server-side for now
			}
		}
	}(self.Ctx)

	return self.Listen()

}

func (self *Server) Listen() error {

	afks := AFK{Map: map[string]int{}}

	r := self.Router

	r.GET("/", func(c *gin.Context) {
		// k := Kobako["index.html"]
		// c.Header("Content-Encoding", "gzip")
		// c.Data(200, k.contentType, k.data)
		c.File(filepath.Join(self.Root, "index.html"))
	})

	r.GET("/js/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		// k := Kobako[filename]
		// c.Header("Content-Encoding", "gzip")
		// c.Data(200, k.contentType, k.data)
		c.File(filepath.Join(self.Root, filename))
	})

	r.GET("/library", func(c *gin.Context) {
		c.JSON(200, self.Library)
	})

	r.GET("/members", func(c *gin.Context) {
		c.JSON(200, self.Service.Members)
	})

	r.POST("/broadcast", func(c *gin.Context) {
		m := NotifyMessage{}
		json.NewDecoder(c.Request.Body).Decode(&m)
		self.BroadcastChan <- m
		c.JSON(200, gin.H{"receivers": self.Service.Members})
	})

	r.GET("/afks", func(c *gin.Context) {
		c.JSON(200, afks.Map)
	})

	r.HEAD("/_afk/:hostname/:secondsaway", func(c *gin.Context) {
		hostname := c.Param("hostname")
		secondsaway, _ := strconv.Atoi(c.Param("secondsaway"))
		afks.Lock()
		defer afks.Unlock()
		afks.Map[hostname] = secondsaway
		c.String(200, "ok")
	})

	r.POST("/_checkout", self.POST_Checkout)

	r.POST("/_update", self.POST_Update)

	r.GET("/_legs", func(c *gin.Context) {
		// k := Kobako["abby.jpg"]
		// c.Header("Content-Encoding", "gzip")
		// c.Data(200, k.contentType, k.data)
		c.File(filepath.Join(self.Root, "abby.jpg"))
	})

	log.Printf("😎 server started: %s :%d [%s]", self.Hostname, self.Port, self.Service.TXTRecord["version"])

	go r.Run(fmt.Sprintf(":%d", self.Port))

	<-self.Ctx.Done()

	return nil

}

func (self *Server) Checkout(cl ClientPayload) CheckoutResponse {

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
			checkout.Checkouts = append(checkout.Checkouts, uuid)
		}
		uuids[uuid] = true
	}

	if len(checkout.Checkouts) > 0 {
		for _, uuid := range checkout.Checkouts {
			log.Printf("✅ [%s] by %s", uuid, cl.Hostname)
		}
	}

	closed, removed := self.Library.DeregisterProjects(cl.Hostname, uuids)
	checkout.Closed = closed

	// Print what was removed/closed
	if len(closed) > 0 || len(removed) > 0 {
		for stat, dd := range map[string][]string{"CLOSED": closed, "REMOVED": removed} {
			for _, uuid := range dd {
				log.Printf("⚠ [%s] [%s] %s", cl.Hostname, stat, uuid)
			}
		}
	}

	return checkout

}
