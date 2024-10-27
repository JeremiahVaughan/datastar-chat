package main

import (
	"log"
	"net/http"
)

func HandleUnexpectedError(w http.ResponseWriter, err error) {
	log.Printf("ERROR: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
