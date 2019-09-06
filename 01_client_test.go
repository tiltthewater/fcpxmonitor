package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func T_FakeLibrary() *FCPLibrary {
	return &FCPLibrary{"Test Project", "/path/to/test.fcpbundle", "1234", map[string]string{}, 0}
}

func T_FakeClient() Client {
	c := NewClient(Service{
		Hostname: "nobody",
		Port:     1234,
		Members:  map[string]string{"other": "http://127.0.0.1:1234"},
	})
	c.Libraries["1234"] = T_FakeLibrary()
	return c
}

func Test_isSame(t *testing.T) {
	c := T_FakeClient()
	newLibs := FCPLibraries{"1234": T_FakeLibrary()}
	if !c.isSame(newLibs) {
		t.Fatal()
	}

	newLibs = FCPLibraries{"3456": T_FakeLibrary()}
	if c.isSame(newLibs) {
		t.Fatal()
	}

	newLibs = FCPLibraries{"1234": T_FakeLibrary()}
	newLibs["1234"].Path = "/new/path/test.fcpbundle"
	if c.isSame(newLibs) {
		t.Fatal()
	}
}

func Test_Client_Can_Be_Marshaled(t *testing.T) {
	c := T_FakeClient()
	s := c.toJSON()
	if s == "" {
		t.Fatal("Failed to marshal struct Client{}")
	}
}

func Test_Unmarshal_Client(t *testing.T) {
	c := T_FakeClient()
	s := c.toJSON()
	if s == "" {
		t.Fatal("Failed to marshal struct Client{}")
	}
	c, err := ClientFromJSON([]byte(s))
	if err != nil {
		t.Fatal(err.Error())
	}
	if c.Hostname != "nobody" {
		t.Fatalf("Unmarshal failed [hostname] %s", c.Hostname)
	}
	if c.Port != 1234 {
		t.Fatalf("Unmarshal failed [port] %d", c.Port)
	}
	_, hasKey := c.Libraries["1234"]
	if !hasKey {
		t.Fatal("Missing library")
	}
}

func Test_Client_Broadcast_Remove_Dead(t *testing.T) {
	c := T_FakeClient()
	// This will timeout, and "other" will be added to the AWOL list
	c.Broadcast()
	_, hasKey := c.AWOL["other"]
	if !hasKey {
		t.Fatal("Expected 'other' to be AWOLed")
	}
	/* Dial back AWOL to two days ago, which will cause it to be removed
	when we broadcast again */
	c.AWOL["other"] = time.Now().AddDate(0, 0, -2)
	c.Broadcast()
	_, hasKey = c.Service.Members["other"]
	if hasKey {
		t.Fatal("Expected 'other' to be removed")
	}
}

func Test_Client_Broadcast(t *testing.T) {

	h := &http.Server{Addr: ":1234"}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)

	c2 := Client{}

	http.HandleFunc("/_checkout", func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		c2, _ = ClientFromJSON(b)
		w.WriteHeader(http.StatusAccepted)
	})

	go h.ListenAndServe()

	c := T_FakeClient()
	c.Broadcast()

	h.Shutdown(ctx)

	if c2.Hostname != "nobody" {
		t.Fatalf("Unmarshal failed [hostname] '%s'", c2.Hostname)
	}
	if c2.Port != 1234 {
		t.Fatalf("Unmarshal failed [port] '%d'", c2.Port)
	}
	_, hasKey := c2.Libraries["1234"]
	if !hasKey {
		t.Fatal("Missing library")
	}

}

func Test_Client_Update(t *testing.T) {

	h := &http.Server{Addr: ":1234"}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)

	update := ProjectUpdate{}

	http.HandleFunc("/_update", func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(b, &update)
		w.WriteHeader(http.StatusOK)
	})

	go h.ListenAndServe()

	c := T_FakeClient()
	t1 := time.Now().Unix()
	c.Update(ProjectUpdate{
		Hostname: "nobody",
		UUID:     "1234",
		Last:     t1,
	})

	time.Sleep(1 * time.Second)

	h.Shutdown(ctx)

	if update.Hostname != "nobody" || update.UUID != "1234" || update.Last != t1 {
		t.Fatal(update)
	}

}

func Test_Notify(t *testing.T) {
	t.SkipNow()
	res := NotifyAlert("Hello World!", "What you want willis?")
	fmt.Println(res)
}
