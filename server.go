package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func NewServer(service Service) Server {
	return Server{
		Hostname:   service.Hostname,
		Port:       service.Port,
		Service:    service,
		LibChan:    make(chan Client, 100),
		UpdateChan: make(chan ProjectUpdate, 100),
		Library:    NewLibrary(),
	}
}

type Server struct {
	Hostname   string
	Port       int
	Library    Library
	Service    Service
	LibChan    chan Client
	UpdateChan chan ProjectUpdate
}

func (self *Server) Start() error {

	ctx, shutdown := context.WithCancel(context.Background())

	go self.Update(ctx)
	go self.Monitor(ctx)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Request-Method", "GET")
	})

	r.GET("/", func(c *gin.Context) {
		k := Kobako["index.html"]
		c.Header("Content-Encoding", "gzip")
		c.Data(200, k.contentType, k.data)
	})

	r.GET("/js/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		k := Kobako[filename]
		c.Header("Content-Encoding", "gzip")
		c.Data(200, k.contentType, k.data)
	})

	r.GET("/library", func(c *gin.Context) {
		c.JSON(200, self.Library)
	})

	r.GET("/_ping", func(c *gin.Context) {
		c.JSON(200, gin.H{self.Hostname: "pong"})
	})

	r.POST("/_checkout", func(c *gin.Context) {
		b, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		d, err := ClientFromJSON(b)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		self.LibChan <- d
		if len(d.Libraries) > 0 {
			uuids_string := ""
			for uuid, _ := range d.Libraries {
				uuids_string = uuids_string + " | " + uuid
			}
			log.Printf("[ACCEPTED] [%s] %s", d.Hostname, uuids_string)
			c.JSON(http.StatusAccepted, gin.H{"ok": uuids_string})
			return
		}
		log.Printf("[ACCEPTED] [%s] No projects open", d.Hostname)
		c.JSON(http.StatusAccepted, gin.H{"ok": "No projects open"})
	})

	r.POST("/_update", func(c *gin.Context) {
		b, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		update := ProjectUpdate{}
		json.Unmarshal(b, &update)
		status, err := self.Library.Update(update)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"ok": fmt.Sprintf("%d", update.Last)})
	})

	r.GET("/_shutdown", func(c *gin.Context) {
		shutdown()
		c.JSON(200, gin.H{"ok": "shutdown"})
	})

	r.GET("/_legs", func(c *gin.Context) {
		k := Kobako["abby.jpg"]
		c.Header("Content-Encoding", "gzip")
		c.Data(200, k.contentType, k.data)
	})

	log.Printf("[INFO] Server started: %s (%d)", self.Hostname, self.Port)

	go r.Run(fmt.Sprintf(":%d", self.Port))

	<-ctx.Done()

	return nil

}

func (self *Server) Update(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
		case c := <-self.LibChan:
			uuids := map[string]bool{}
			for uuid, lib := range c.Libraries {
				err := self.Library.CheckoutProject(uuid, lib.Name, c.Hostname, lib.Path, lib.Info)
				if err != nil {
					// TODO: Multiple checkouts, notify clients!
					log.Println("[ERROR] " + err.Error())
				}
				uuids[uuid] = true
			}
			removed := self.Library.DeregisterProjects(c.Hostname, uuids)
			if len(removed) > 0 {
				log.Printf("[CLOSED] %s", strings.Join(removed, " | "))
			}
		case u := <-self.UpdateChan:
			noop(u)
		}
	}
}

func (self *Server) Monitor(ctx context.Context) {
	for {
		// No use server-side for now
		<-self.Service.BroadcastChan
	}
}

func ClientFromJSON(b []byte) (Client, error) {
	c := Client{}
	err := json.Unmarshal(b, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}
