package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/adrianloh/echien"
)

var (
	re_outputs  = regexp.MustCompile(`(?i)^outputs?$`)
	re_versions = regexp.MustCompile(`(?i)[ _-]v(\d+)[a-z]?[. _-].*(mov|mp4|m4v)`)
)

type FCPLibrary struct {
	Name string            `json:"name"`
	Path string            `json:"path"`
	UUID string            `json:"uuid"`
	Info map[string]string `json:"info"`
	Last int64             `json:"last"`
}

var (
	re_fcpbundle   = regexp.MustCompile(`^(.+\.fcpbundle)\/.*CurrentVersion.fcpevent.*`)
	re_bundle_uuid = regexp.MustCompile(`\w{8}-\w{4}-\w{4}-\w{4}-\w{12}`)
)

func NewFCPProject(bundlePath string) (lib FCPLibrary, err error) {
	_, bundle := path.Split(bundlePath)
	lib = FCPLibrary{
		Name: bundle,
		Path: bundlePath,
		UUID: "0000",
		Last: 0,
		Info: map[string]string{},
	}
	lib.UUID, err = GetLibraryUUID(lib.Path)
	version_mtime := FindLatestVersion(bundlePath)
	if version_mtime[0] != "" {
		lib.Info["version"] = version_mtime[0]
		lib.Info["version_mtime"] = version_mtime[1]
	}
	return lib, err
}

func GetLatestVersionName(filenames []string) (highest string) {
	version := 0
	for _, fn := range filenames {
		m := re_versions.FindAllStringSubmatch(fn, -1)
		if len(m) > 0 {
			i, err := strconv.Atoi(m[0][1])
			if err == nil {
				if i > version {
					version = i
					highest = fn
				}
			}
		}
	}
	return highest
}

func FindLatestVersion(bundlePath string) [2]string {

	version_mtime := [2]string{}
	versions := []string{}
	mtimes := map[string]string{}

	parent, _ := filepath.Split(bundlePath)
	files, _ := ioutil.ReadDir(parent)

	for _, fifo := range files {
		if fifo.IsDir() && re_outputs.MatchString(fifo.Name()) {
			outputs_folder := filepath.Join(parent, fifo.Name())
			files, _ := ioutil.ReadDir(outputs_folder)
			for _, fifo := range files {
				if !fifo.IsDir() && re_versions.MatchString(fifo.Name()) {
					versions = append(versions, fifo.Name())
					mtimes[fifo.Name()] = strconv.FormatInt(fifo.ModTime().Unix(), 10)
				}
			}
			if len(versions) > 0 {
				fn := GetLatestVersionName(versions)
				version_mtime[0] = fn
				version_mtime[1] = mtimes[fn]
				return version_mtime
			}
			break
		}
	}
	return version_mtime
}

func GetLibraryUUID(libPath string) (string, error) {
	uuid := ""
	settingsPlist := path.Join(libPath, "Settings.plist")
	e, err := echien.Open(settingsPlist)
	if err != nil {
		return uuid, err
	}
	els := e.Find("string")
	if len(els) > 0 {
		for _, el := range els {
			text := el.GetAttribute("text")
			if re_bundle_uuid.MatchString(text) {
				uuid = text
				break
			}
		}
	} else {
		return uuid, errors.New("No <string> tags found: " + libPath)
	}
	if len(uuid) != 36 {
		return uuid, errors.New(fmt.Sprintf("Weird uuid in <string> tag: %s | %s", uuid, libPath))
	}
	return uuid, nil
}

func BundlePath(fp string) (str string) {
	m := re_fcpbundle.FindAllStringSubmatch(fp, -1)
	if len(m) == 0 {
		return str
	}
	if strings.Contains(fp, "/private/") {
		return str
	}
	if strings.Contains(fp, "Final Cut Backups") {
		return str
	}
	if strings.Contains(fp, "__Temp") {
		return str
	}
	return "/" + m[0][1]
}

func GetOpenFCPLibraries() (libs FCPLibraries, errs []error) {

	libs = FCPLibraries{}
	errs = make([]error, 0)

	stdout, err := exec.Command("lsof", "-F", "n0", "-wbc", "Final Cut Pro").Output()
	if err != nil {
		if err.Error() == "exit status 1" {
			err = errors.New("FCPX is not active")
		}
		errs = append(errs, err)
		return libs, errs
	} else if len(stdout) == 0 {
		return libs, errs
	}
	captured := map[string]bool{}
	lines := strings.Split(string(stdout), "\n")
	for _, line := range lines {
		ss := strings.Split(line, "\x00")
		if len(ss) > 1 {
			fp := strings.TrimLeft(ss[1], "n/")
			bundle := BundlePath(fp)
			if bundle != "" && !captured[bundle] {
				captured[bundle] = true
				lib, err := NewFCPProject(bundle)
				if err != nil {
					errs = append(errs, err)
				}
				_, hasKey := libs[lib.UUID]
				if hasKey {
					continue
				}
				libs[lib.UUID] = &lib
			}
		}
	}
	return libs, errs
}
