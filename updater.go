package main

import (
	"fmt"
	"log"
	"net/http"

	"adrian.wtf/selfupdate"
)

func NewUpdater(runDir string, port int) (selfupdate.SelfUpdate, error) {

	updater, err := selfupdate.New(runDir)
	if err != nil {
		return updater, err
	}

	go func(port int) {
		version := <-updater.ChangeChan
		url := fmt.Sprintf("http://localhost:%d/_shutdown", port)
		log.Println("🤞 [UPDATE] version: " + version[0:7])
		res, err := http.Get(url)
		if err != nil || res.StatusCode != http.StatusOK {
			LogError("[UPDATE] Failed")
		}
	}(port)

	return updater, nil

}
