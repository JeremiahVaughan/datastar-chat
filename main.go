package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/JeremiahVaughan/datastar-chat/ui_util"
	"github.com/fsnotify/fsnotify"
)

var localMode bool

var watcher *fsnotify.Watcher
var tl *ui_util.TemplateLoader

func main() {
	log.Println("starting chat app")
	localMode = os.Getenv("LOCAL_MODE") == "true"
	var err error
	templates := []ui_util.HtmlTemplate{
		{
			Name: "base",
		},
	}
	tl, err = ui_util.NewTemplateLoader(
		"templates/base",
		"templates/overrides",
		templates,
		localMode,
	)
	if err != nil {
		log.Fatalf("error, when ui_util.NewTemplateLoader() for main(). Error: %v", err)
	}

	err = initConfig()
	if err != nil {
		log.Fatalf("error, when initConfig() for main(). Error: %v", err)
	}
	mux := http.NewServeMux()

	// endpoints key is endpoint address
	endpoints := map[string]http.HandlerFunc{
		"/":            handleHome,
		"/chatFeed":    handleChatFeed,
		"/hotreload":   tl.HandleHotReload,
		"/sendMessage": handleSendMessage,
	}
	for k, v := range endpoints {
		mux.Handle(k, v)
	}
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("error, when setting up http server for main(). Error: %v", err)
	}
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	input, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("error, when reading response body. Error: %v", err), http.StatusInternalServerError)
		return
	}
	sendHeaders(w)
	output := fmt.Sprintf("Your input: %s, is %d long.", input, len(input))
	frag := fmt.Sprintf(`<div id="output">%s</div>`, output)
	err = sendSSE(w, "", "", frag, true)
	if err != nil {
		http.Error(w, fmt.Sprintf("error, when sending response 1 for handlePut(). Error: %v", err), http.StatusInternalServerError)
		return
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	sendHeaders(w)
	output := "Backend Data: stuff, stuff, and more stuff"
	frag := fmt.Sprintf(`<div id="output2">%s</div>`, output)
	err := sendSSE(w, "", "", frag, true)
	if err != nil {
		http.Error(w, fmt.Sprintf("error, when sending response 1 for handleGet(). Error: %v", err), http.StatusInternalServerError)
		return
	}

	time.Sleep(1 * time.Second)
	frag = `<div id="output3">Check this out!</div>;`
	err = sendSSE(w, "", "", frag, true)
	if err != nil {
		http.Error(w, fmt.Sprintf("error, when sending response 2 for handleGet(). Error: %v", err), http.StatusInternalServerError)
		return
	}
}

type BaseData struct {
	LocalMode bool
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	err := tl.GetTemplateGroup("base").ExecuteTemplate(w, "base", BaseData{LocalMode: true})
	if err != nil {
		http.Error(w, fmt.Sprintf("error, when serving base.html. Error: %v", err), http.StatusInternalServerError)
		return
	}
}

func sendHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "Keep-Alive")
	w.(http.Flusher).Flush()
}

func sendSSE(w http.ResponseWriter, selector string, mergeType string, fragment string, end bool) error {
	_, err := w.Write([]byte("event: datastar-fragment\n"))
	if err != nil {
		return fmt.Errorf("error, when writing first line of sendSSE(). Error: %v", err)
	}

	if selector != "" {
		_, err = fmt.Fprintf(w, "data: selector %s\n", selector)
		if err != nil {
			return fmt.Errorf("error, when writing second line of sendSSE(). Error: %v", err)
		}
	}

	if mergeType != "" {
		_, err = fmt.Fprintf(w, "data: merge %s\n", mergeType)
		if err != nil {
			return fmt.Errorf("error, when writing third line of sendSSE(). Error: %v", err)
		}
	}

	if fragment != "" {
		_, err = fmt.Fprintf(w, "data: fragment %s\n\n", fragment)
		if err != nil {
			return fmt.Errorf("error, when writing fourth line of sendSSE(). Error: %v", err)
		}
	}

	if end {
		w.(http.Flusher).Flush()
	}
	return nil
}
