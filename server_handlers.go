package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

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

	update := ProjectUpdate{}
	err := json.NewDecoder(c.Request.Body).Decode(&update)
	if err != nil {
		LogError(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	self.Library.Lock()
	defer self.Library.Unlock()

	status, err := self.Library.Update(update)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"ok": fmt.Sprintf("%d", update.Last)})

}
