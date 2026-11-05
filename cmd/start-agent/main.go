package main

import (
	"log"

	"github.com/CiaranW005/pc-agent/internal/connection"
	"github.com/CiaranW005/pc-agent/internal/executor"
	"github.com/CiaranW005/pc-agent/internal/router"
)

func main() {
	exec := executor.NewExecutor()
	manager := connection.NewManager()
	handler := router.NewRouter(exec, manager.Tracker(), manager.SendTracked)

	defer manager.CloseSession()

	// Listen on all interfaces for the ESP32's TCP connection.
	// StartSession blocks while accepting and handling connections.
	err := manager.StartSession(":9090", handler.Handle)
	if err != nil {
		log.Fatal(err)
	}
}
