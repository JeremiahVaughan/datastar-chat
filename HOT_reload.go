package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var UiFilesBeChangin chan fsnotify.Event = make(chan fsnotify.Event)

func handleHotreload(w http.ResponseWriter, r *http.Request) {
	sendHeaders(w)
	i := 0
	retrySleep := 250 * time.Millisecond
	var lastEventTime int64
	var events strings.Builder
	for {
		event := <-UiFilesBeChangin
		// debounce 5 miliseconds
		debounceTimeMilli := int64(5)
		if !isChangedEvent(event) {
			continue
		}
		_, err := events.WriteString(fmt.Sprintf("name=%s@time=%d", event.Name, time.Now().UnixMilli()))
		if err != nil {
			log.Fatalf("error, when writing event to string builder. Error: %v", err)
		}
		if lastEventTime != 0 && lastEventTime+debounceTimeMilli < time.Now().UnixMilli() {
			continue
		}
		lastEventTime = time.Now().UnixMilli()
		go func(index int) {
			// wait till all file operations have settled, this ensures the files are in the desired state when we parse them
			time.Sleep(time.Duration(debounceTimeMilli) * time.Millisecond)
			err2 := parseDevTemplates()
			if err2 != nil {
				err2 = fmt.Errorf("error, when parseDevTemplates() for handleHotreload(). Error: %v", err2)
				http.Error(w, err2.Error(), http.StatusInternalServerError)
				log.Fatal(err2.Error())
			}
			evt := fmt.Sprintf(
				"id:%X\nretry:%d\ndata:%s\n\n",
				index, retrySleep, events.String(),
			)
			events.Reset()
			w.Write([]byte(evt))
			w.(http.Flusher).Flush()
			index++
		}(i)
	}
}

func isDir(path string) bool {
	i, err := os.Stat(path)
	if err != nil {
		return false
	}
	return i.IsDir()
}

func isChangedEvent(ev fsnotify.Event) bool {
	return ev.Op&fsnotify.Create == fsnotify.Create ||
		ev.Op&fsnotify.Write == fsnotify.Write ||
		ev.Op&fsnotify.Remove == fsnotify.Remove
}

func watchDemFiles(watcher *fsnotify.Watcher) error {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Println("event watcher closed")
				return nil
			}
			UiFilesBeChangin <- event
		case err, ok := <-watcher.Errors:
			if !ok {
				log.Println("error watcher closed")
				return nil
			}
			return fmt.Errorf("error, watching html files for changes. Error: %v", err)
		}
	}
}
