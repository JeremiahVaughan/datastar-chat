package main

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

var ns *server.Server

func initConfig() error {
	err := initNats()
	if err != nil {
		return fmt.Errorf("error, when initNats() for initConfig(). Error: %v", err)
	}
	return nil
}

func initNats() error {
	// Configure NATS Server options
	opts := &server.Options{
		Port: -1, // Let the server pick an available port
		// You can set other options here (e.g., authentication, clustering)
	}

	// Create a new NATS server instance
	var err error
	ns, err = server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("error, when creating NATS server. Error: %v", err)
	}

	// Start the server in a separate goroutine
	go ns.Start()

	// Ensure the server has started
	if !ns.ReadyForConnections(10 * time.Second) {
		return errors.New("error, NATS Server didn't start in time")
	}

	// Retrieve the server's listen address
	addr := ns.Addr()
	var port int
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		port = tcpAddr.Port
	} else {
		return fmt.Errorf("error, filed to get nats server port")
	}
	fmt.Printf("NATS server is running on port %d\n", port)
	return nil
}

func connectToNats() (*nats.Conn, error) {
	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		return nil, fmt.Errorf("error, when connecting to NATS server. Error: %v", err)
	}
	return nc, nil
}
