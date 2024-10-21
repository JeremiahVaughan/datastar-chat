package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fsnotify/fsnotify"
)

var UiFilesBeChangin chan bool = make(chan bool)

func handleHotreload(w http.ResponseWriter, r *http.Request) {
	sendHeaders(w)
	i := 0
	retrySleep := 250 * time.Millisecond
	var lastEventTime int64
	for {
		<-UiFilesBeChangin
		// debounce 5 miliseconds
		debounceTimeMilli := int64(5)
		if lastEventTime != 0 && lastEventTime+debounceTimeMilli < time.Now().UnixMilli() {
			continue
		}
		lastEventTime = time.Now().Unix()
		// wait till all file operations have settled, this ensures the files are in the desired state when we parse them
		time.Sleep(time.Duration(debounceTimeMilli) * time.Millisecond)
		err := parseDevTemplates()
		if err != nil {
			err = fmt.Errorf("error, when parseDevTemplates() for handleHotreload(). Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatal(err.Error())
		}
		evt := fmt.Sprintf(
			"id:%X\nretry:%d\ndata:%x\n\n",
			i, retrySleep, i,
		)
		w.Write([]byte(evt))
		w.(http.Flusher).Flush()
		i++
	}
}

func watchDemFiles(watcher *fsnotify.Watcher) error {
	for {
		select {
		case _, ok := <-watcher.Events:
			if !ok {
				log.Println("event watcher closed")
				return nil
			}
			UiFilesBeChangin <- true
		case err, ok := <-watcher.Errors:
			if !ok {
				log.Println("error watcher closed")
				return nil
			}
			return fmt.Errorf("error, watching html files for changes. Error: %v", err)
		}
	}
}
