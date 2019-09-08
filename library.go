package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

/*
                    |- Name
	Library[uuid] --|- Checkouts[hostname] --|- Path
                                             |- Last
*/

type Project struct {
	Name      string               `json:"name"`
	Info      map[string]string    `json:"info"`
	Checkouts map[string]*Checkout `json:"checkouts"`
}

type Checkout struct {
	Path string `json:"path"`
	Last int64  `json:"last"`
}

type ProjectUpdate struct {
	Hostname string `json:"hostname"`
	UUID     string `json:"uuid"`
	Last     int64  `json:"last"`
}

func NewLibrary() Library {
	return Library{
		Projects: map[string]*Project{},
	}
}

type Library struct {
	sync.Mutex
	Projects map[string]*Project `json:"library"`
}

func (self *Library) HasProject(uuid string) bool {
	_, hasKey := self.Projects[uuid]
	return hasKey
}

func (self *Library) CheckoutProject(uuid, name, host, path string, info map[string]string) error {
	if !self.HasProject(uuid) {
		self.Projects[uuid] = &Project{
			Name: name,
			Info: info,
			Checkouts: map[string]*Checkout{
				host: &Checkout{
					Path: path,
					Last: 0,
				},
			},
		}
		return nil
	}
	checkouts := self.Projects[uuid].Checkouts
	cc, hasKey := checkouts[host]
	self.Projects[uuid].Info = info
	if !hasKey {
		checkouts[host] = &Checkout{
			Path: path,
			Last: 0,
		}
	} else if cc.Path != path {
		cc.Path = path
	}
	otherhosts := []string{}
	for currHost, _ := range checkouts {
		if currHost != host {
			otherhosts = append(otherhosts, currHost)
		}
	}
	if len(otherhosts) != 0 {
		msg := fmt.Sprintf(`Project '%s'(%s) multiple checkouts:\t%s|%s`, name, uuid, host, strings.Join(otherhosts, "|"))
		return errors.New(msg)
	}
	return nil
}

func (self *Library) Update(update ProjectUpdate) (int, error) {
	if !self.HasProject(update.UUID) {
		return http.StatusNotFound, errors.New("Project does not exist: " + update.UUID)
	}
	chk, hasKey := self.Projects[update.UUID].Checkouts[update.Hostname]
	if !hasKey {
		return http.StatusForbidden, errors.New("No previous checkout from " + update.Hostname)
	}
	chk.Last = Latest(update.Last, chk.Last)
	return 0, nil
}

func (self *Library) DeregisterProjects(host string, openedProjects map[string]bool) (closed, removed []string) {
	closed = []string{}
	removed = []string{}
	for uuid, project := range self.Projects {
		_, hasKey := project.Checkouts[host]
		if hasKey && !openedProjects[uuid] {
			delete(project.Checkouts, host)
			closed = append(closed, uuid)
		}
	}
	for uuid, project := range self.Projects {
		if len(project.Checkouts) == 0 {
			delete(self.Projects, uuid)
			removed = append(removed, uuid)
		}
	}
	return closed, removed
}
