package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func Test_Service(t *testing.T) {

	port := 1234

	h := &http.Server{Addr: ":1234"}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)

	http.HandleFunc("/_ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"ok":"pong"}`)
	})

	go h.ListenAndServe()

	client_service := NewService("megan", port, SERVICE_CLIENT, nil)
	go client_service.Start()
	server_service := NewService("martha", port, SERVICE_SERVER, nil)
	go server_service.Start()
	time.Sleep(5 * time.Second)

	server_service.ExitChan <- true
	client_service.ExitChan <- true
	time.Sleep(5 * time.Second)

	_, hasKey := client_service.Members["martha"]
	if !hasKey {
		t.Fatal("Client did not find server")
	}

	_, hasKey = server_service.Members["megan"]
	if !hasKey {
		t.Fatal("Server did not find client")
	}

	h.Shutdown(ctx)

}
