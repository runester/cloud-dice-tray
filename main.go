package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/runester/cloud-dice-tray/internal/config"
	appweb "github.com/runester/cloud-dice-tray/internal/web"
)

func main() {
	configPath := flag.String("config", config.DefaultPath, "path to YAML configuration file")
	flag.Parse()

	configuration, err := config.Load(*configPath)
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(1)
	}

	application, err := appweb.New()
	if err != nil {
		log.Printf("initialize application: %v", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              configuration.Server.ListenAddress,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("cloud-dice-tray listening on %s", configuration.Server.ListenAddress)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
