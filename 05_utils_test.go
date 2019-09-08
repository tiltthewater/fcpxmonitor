package main

import (
	"fmt"
	"testing"
	"time"
)

func Test_Get_Latest_Version(t *testing.T) {

	cases := []string{
		"Pepsi Generation V1.mp4",
		"Pepsi Generation v1 .mp4",
		"Pepsi Generation v1 .mp4",
		"Pepsi Generation v1 .mp4",
	}

	for _, testName := range cases {
		if !re_versions.MatchString(testName) {
			t.Fatal(testName)
		}
	}

	versions := []string{
		"Pepsi Generation V1.mp4",
		"Pepsi Generation V2.mp4",
		"Pepsi Generation V3.mp4",
		"Pepsi Generation V4.mp4",
	}
	if v := GetLatestVersionName(versions); v != "Pepsi Generation V4.mp4" {
		t.Fatal(v)
	}

}

func Test_USB_Activity(t *testing.T) {
	fmt.Println("🖐 No touching of mouse or keyboard please...")
	time.Sleep(7 * time.Second)
	elapsed := LastUSBActivity()
	if elapsed < 4 {
		t.Fatal(elapsed)
	}
}
