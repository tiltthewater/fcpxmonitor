package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	re_idletime = regexp.MustCompile(`"HIDIdleTime" = (\d+)`)
)

func LogError(msg string) {
	callInfo := ""
	pc, file, lineNo, ok := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash
	_, sourceFile := filepath.Split(file)
	if ok {
		callInfo = fmt.Sprintf("[ %s ] %s line %d", funcName[lastDot+1:], sourceFile, lineNo)
	}
	log.Printf("🔴 %s\n\t==> %s", msg, callInfo)
}

func LogFatal(msg string) {
	callInfo := ""
	pc, file, lineNo, ok := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash
	_, sourceFile := filepath.Split(file)
	if ok {
		callInfo = fmt.Sprintf("[ %s ] %s line %d", funcName[lastDot+1:], sourceFile, lineNo)
	}
	log.Fatalf("⛔️ %s\n\t==> %s", msg, callInfo)
}

func LogWarning(msg ...string) {
	log.Println("⚠️ ", msg)
}

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

func LastUSBActivity() int64 {
	timings := make([]int64, 0)
	stdout, _ := exec.Command("ioreg", "-c", "IOHIDSystem").Output()
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		if re_idletime.MatchString(line) {
			m := re_idletime.FindAllStringSubmatch(line, -1)
			t, err := strconv.ParseInt(m[0][1], 10, 64)
			if err == nil {
				timings = append(timings, t)
			}
		}
	}
	if len(timings) == 0 {
		return -1
	}
	return timings[0] / 1000000000
}
