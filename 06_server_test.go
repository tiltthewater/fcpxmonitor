package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func Test_Lag(t *testing.T) {

	t1 := time.Now()
	time.Sleep(500 * time.Millisecond)
	l := Lag(t1)
	if !(l > 500 && l < 510) {
		t.Fatal(l)
	}

}

func Test_Alive(t *testing.T) {

	h := &http.Server{Addr: ":1234"}

	http.HandleFunc("/_pong", func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(T{time.Now()})
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	go h.ListenAndServe()

	s := NewServer(Service{
		Hostname: "nobody",
		Port:     1234,
		Members:  map[string]string{"other": "http://127.0.0.1:1234"},
	})

	s.CheckMembersAlive()

	time.Sleep(500 * time.Millisecond)

	h.Shutdown(context.TODO())

	time.Sleep(500 * time.Millisecond)

	l := s.ResponseTimes["other"]
	if !(l > 500 && l < 510) {
		t.Fatal(l)
	}

}
