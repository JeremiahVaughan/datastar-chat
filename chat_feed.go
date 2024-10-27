package main

import (
	"fmt"
	"net/http"
	"time"
)

func handleChatFeed(w http.ResponseWriter, _ *http.Request) {
	sendHeaders(w)
	nc, err := connectToNats()
	if err != nil {
		err = fmt.Errorf("error, when connectToNats() for handleChatFeed(). Error: %v", err)
		HandleUnexpectedError(w, err)
		return
	}
	sub, err := nc.SubscribeSync(MessageFeed)
	if err != nil {
		err = fmt.Errorf("error, when subscribing to messages for handleChatFeed(). Error: %v", err)
		HandleUnexpectedError(w, err)
		return
	}
	for {
		msg, err := sub.NextMsg(300 * time.Hour) // the duration is required but doesn't matter for our use-case so the 300 hours is just arbitrary
		if err != nil {
			err = fmt.Errorf("error, when retrieving next message for handleChatFeed(). Error: %v", err)
			HandleUnexpectedError(w, err)
			return
		}
		frag := `<div id="store" data-store='{ "message": "" }'></div>`
		err = sendSSE(w, "#store", "", frag, false)
		if err != nil {
			err = fmt.Errorf("error, when sending reset message response for handleChatFeed(). Error: %v", err)
			HandleUnexpectedError(w, err)
			return
		}
		frag = fmt.Sprintf(`<div>%s</div>`, msg.Data)
		err = sendSSE(w, "#output", "append", frag, true)
		if err != nil {
			err = fmt.Errorf("error, when sending new chat messages response for handleChatFeed(). Error: %v", err)
			HandleUnexpectedError(w, err)
			return
		}
	}
}
