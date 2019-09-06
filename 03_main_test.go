package main

import (
	"context"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
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

func Test_Get_Library_UUID(t *testing.T) {
	uuid, err := GetLibraryUUID(path.Join(rundir, TEST_PROJECT_BUNDLE))
	if err != nil {
		t.Fatal(err.Error())
	}
	if uuid != TEST_PROJECT_UUID {
		t.Fatalf("Wrong uuid: %s\n", uuid)
	}
}

func Test_Get_Open_Libraries(t *testing.T) {
	fakeFCP := path.Join(rundir, "Final Cut Pro")
	go exec.Command(fakeFCP).Run()
	time.Sleep(time.Second)
	libs, errs := GetOpenFCPLibraries()
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	_, hasKey := libs[TEST_PROJECT_UUID]
	if !hasKey {
		t.Fatal(libs)
	}
}

func Test_FSWatch(t *testing.T) {
	c := make(chan FCPLibrary, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	i := 0
	go func() {
		for {
			s := <-c
			log.Printf("[%s] %d", s.UUID, s.Last)
			i += 1
			if i == 5 {
				cancel()
			}
		}
	}()
	go func() {
		fakefile := path.Join(project_path, "_CurrentVersion.fcpevent")
		for i := 0; i < 5; i++ {
			time.Sleep(time.Second)
			data := []byte(time.Now().Format(time.RFC1123))
			ioutil.WriteFile(fakefile, data, 0666)
		}
	}()
	go WatcherPath([]string{rundir}, c, ctx)
	<-ctx.Done()
	if i != 5 {
		t.Errorf("Did not receive 5 change messages: %d\n", i)
	}
}

func Test_USBHeartbeat(t *testing.T) {
	t.SkipNow()
	time.Sleep(4 * time.Second)
	elapsed := UsbHeartbeat()
	if elapsed < 4 {
		t.Fatal(elapsed)
	}
}
