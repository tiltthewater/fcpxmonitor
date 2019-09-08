package main

import (
	"context"
	"log"
	"time"
)

func WatcherOpenLibraries(libsChan chan FCPLibraries, tickChan <-chan time.Time, ctx context.Context) {
	lastMessage := ""
mainLoop:
	for {
		select {
		case <-ctx.Done():
			break mainLoop
		case <-tickChan:
			libs, errs := GetOpenFCPLibraries()
			if len(errs) > 0 {
				for _, err := range errs {
					msg := "[WARNING] " + err.Error()
					if msg != lastMessage {
						log.Println(msg)
						lastMessage = msg
					}
				}
			} else {
				lastMessage = ""
			}
			libsChan <- libs
		}
	}
	LogWarning("[STOP] Library watch")
}
