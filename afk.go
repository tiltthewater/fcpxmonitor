package main

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	re_idletime = regexp.MustCompile(`"HIDIdleTime" = (\d+)`)
)

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
