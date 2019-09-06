package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func NewHTTPTimeoutClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 5,
	}
}

func Latest(unixT1 int64, unixT2 int64) int64 {
	t1 := time.Unix(unixT1, 0)
	t2 := time.Unix(unixT2, 0)
	if t1.After(t2) {
		return unixT1
	}
	return unixT2
}

func connection_refused(err error) bool {
	return strings.Contains(err.Error(), "connection refused")
}

func NotifyAlert(title, message string) bool {
	s := fmt.Sprintf(`display alert "%s" message "%s"`, title, message)
	exec.Command("osascript", "-e", s).Output()
	return true
}
