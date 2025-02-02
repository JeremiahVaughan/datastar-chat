package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Debouncer struct {
	delay   time.Duration
	timer   *time.Timer
	mu      sync.Mutex
	started bool
}

func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay: delay,
	}
}

func (d *Debouncer) Debounce(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// If a timer is already running, stop it
	if d.timer != nil {
		d.timer.Stop()
	}

	// Start a new timer that will call the function after the delay
	d.timer = time.AfterFunc(d.delay, func() {
		f()
		d.mu.Lock()
		d.timer = nil // Allow the next function call to use a new timer
		d.mu.Unlock()
	})
}

const fileEventsBufferSize = 30

var UiFilesBeChangin chan fsnotify.Event = make(chan fsnotify.Event, fileEventsBufferSize)
var UiFilesBeChanginTimes chan int64 = make(chan int64, fileEventsBufferSize)

func handleHotreload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// watch for events and enrich with current time
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-UiFilesBeChangin:
				currentTime := time.Now().UnixMilli()
				if isChangedEvent(event) {
					log.Printf(
						"file change event detected. File: %s. Operation: %s. Time: %d",
						event.Name,
						event.Op.String(),
						currentTime,
					)
					log.Printf("starting channel at %d", time.Now().UnixMilli())
					UiFilesBeChanginTimes <- currentTime
					log.Printf("draining channel at %d", time.Now().UnixMilli())
				} else {
					log.Printf(
						"file non change event detected. File: %s. Operation: %s. Time: %d",
						event.Name,
						event.Op.String(),
						currentTime,
					)
				}
			}
		}
	}()

	sendHeaders(w)
	i := 0
	retrySleep := 250 * time.Millisecond
	// wait till all file operations have settled, this ensures the
	// files are in the desired state when we parse them
	// debounce 50 miliseconds, as it seems file events are around 6 miliseconds apart and there are 3-4 each time
	db := NewDebouncer(50 * time.Millisecond)
	sb := strings.Builder{}
	for {
		select {
		case <-ctx.Done():
			return
		case eventTime := <-UiFilesBeChanginTimes:
			db.Debounce(func() {
				_, err := sb.WriteString(fmt.Sprintf("name=%d@time=%d", eventTime, time.Now().UnixMilli()))
				if err != nil {
					log.Fatalf("error, when writing event to string builder. Error: %v", err)
				}

				err2 := parseDevTemplates()
				if err2 != nil {
					err2 = fmt.Errorf("error, when parseDevTemplates() for handleHotreload(). Error: %v", err2)
					// http.Error(w, err2.Error(), http.StatusInternalServerError) // not returning the error because in golang you can't set the status code more than once in a single call.
					// And this is a long running call so it is likely to happen more than once
					log.Fatal(err2.Error())
				}
				evt := fmt.Sprintf(
					"id:%X\nretry:%d\ndata:%s\n\n",
					i, retrySleep, sb.String(),
				)
				w.Write([]byte(evt))
				log.Printf("message sent: %d", eventTime)
				w.(http.Flusher).Flush()
				i++
			})
		}
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
