package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
	log.Printf("🤬 %s\n\t==> %s", msg, callInfo)
}

func LogWarning(msg string) {
	log.Printf("⚠️  " + msg)
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

func IsConnectionRefused(err error) bool {
	return strings.Contains(err.Error(), "connection refused")
}
