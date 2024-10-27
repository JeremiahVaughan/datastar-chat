package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Message struct {
	Message string `json:"message"`
}

var MessageFeed = "message_feed"

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	defer w.(http.Flusher).Flush()
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("error, when reading request body for handleSendMessage(). Error: %v", err)
		HandleUnexpectedError(w, err)
		return
	}
	sendHeaders(w)
	m := Message{}
	err = json.Unmarshal(requestBody, &m)
	if err != nil {
		err = fmt.Errorf("error, when unmarshalling request body for handleSendMessage(). Error: %v", err)
		HandleUnexpectedError(w, err)
		return
	}
	if m.Message == "" {
		return
	}
	nc, err := connectToNats()
	if err != nil {
		err = fmt.Errorf("error, when connectToNats() for handleSendMessage(). Error: %v", err)
		HandleUnexpectedError(w, err)
		return
	}
	defer nc.Close()

	err = nc.Publish(MessageFeed, []byte(m.Message))
	if err != nil {
		err = fmt.Errorf("error, when attempting to publish the message for handleSendMessage(). Error: %v", err)
		HandleUnexpectedError(w, err)
		return
	}
}
