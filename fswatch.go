package main

import (
	"context"
	"log"
	"time"

	"github.com/fsnotify/fsevents"
)

var noteDescription = map[fsevents.EventFlags]string{
	fsevents.MustScanSubDirs: "MustScanSubdirs",
	fsevents.UserDropped:     "UserDropped",
	fsevents.KernelDropped:   "KernelDropped",
	fsevents.EventIDsWrapped: "EventIDsWrapped",
	fsevents.HistoryDone:     "HistoryDone",
	fsevents.RootChanged:     "RootChanged",
	fsevents.Mount:           "Mount",
	fsevents.Unmount:         "Unmount",

	fsevents.ItemCreated:       "Created",
	fsevents.ItemRemoved:       "Removed",
	fsevents.ItemInodeMetaMod:  "InodeMetaMod",
	fsevents.ItemRenamed:       "Renamed",
	fsevents.ItemModified:      "Modified",
	fsevents.ItemFinderInfoMod: "FinderInfoMod",
	fsevents.ItemChangeOwner:   "ChangeOwner",
	fsevents.ItemXattrMod:      "XAttrMod",
	fsevents.ItemIsFile:        "IsFile",
	fsevents.ItemIsDir:         "IsDir",
	fsevents.ItemIsSymlink:     "IsSymLink",
}

func WatcherPath(pathsToWatch []string, updateChan chan FCPLibrary, ctx context.Context) {
	es := &fsevents.EventStream{
		Paths:   pathsToWatch,
		Latency: 1000 * time.Millisecond,
		// Device:  dev,
		Flags: fsevents.FileEvents | fsevents.WatchRoot}
	es.Start()
	ec := es.Events
	go func() {
		for msg := range ec {
			for _, event := range msg {
				// Note: event.Path doesn't have a starting slash
				bundle := BundlePath(event.Path)
				if bundle != "" {
					lib, err := NewFCPProject(bundle)
					if err != nil {
						log.Println("[WARNING] [UPDATE] " + err.Error())
					} else {
						lib.Last = time.Now().Unix()
						updateChan <- lib
					}
				}
			}
		}
	}()
	<-ctx.Done()
	es.Stop()
	LogWarning("[STOP] Filesystem watch")
}
