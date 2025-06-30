package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dantdj/esxi-manager/internal/esxi"
	"github.com/dantdj/esxi-manager/internal/schedule"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var retryDelay int = 30
var retryCount int = 5

//go:embed web/dist
var reactDist embed.FS

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("no .env file found, relying on environment variables")
	}

	connection := esxi.Connection{
		Username:   os.Getenv("ESXI_USER"),
		Password:   os.Getenv("ESXI_PASS"),
		URL:        os.Getenv("ESXI_URL"),
		MACAddress: os.Getenv("ESXI_MAC"),
	}

	if connection.Username == "" || connection.Password == "" || connection.URL == "" || connection.MACAddress == "" {
		log.Fatal().Msg("missing one or more required environment variables (ESXI_USER, ESXI_PASS, ESXI_URL, ESXI_MAC)")
	}

	manage := flag.Bool("manage", true, "Enable or disable the ESXi power-on/power-off schedule")
	port := flag.String("port", "8080", "port to start the webserver on")
	flag.Parse()

	if *manage {
		go manageEsxiServer(connection)
	}

	// Get the dist filesystem
	distFS, err := fs.Sub(reactDist, "web/dist")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get sub-filesystem")
	}

	// Serve all static assets (CSS, JS, etc.)
	http.Handle("/assets/", http.FileServer(http.FS(distFS)))

	// SPA fallback - serve index.html for all non-API routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip API routes
		if strings.HasPrefix(path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file if it exists, otherwise serve index.html
		if path != "/" {
			if file, err := distFS.Open(strings.TrimPrefix(path, "/")); err == nil {
				file.Close()
				http.ServeFileFS(w, r, distFS, strings.TrimPrefix(path, "/"))
				return
			}
		}

		// Serve index.html for SPA routing
		http.ServeFileFS(w, r, distFS, "index.html")
	})

	http.HandleFunc("/api/isalive", func(w http.ResponseWriter, r *http.Request) {
		isAlive := connection.ServerReachable()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"is_alive": isAlive})
	})
	http.HandleFunc("/api/turnoff", func(w http.ResponseWriter, r *http.Request) {
		go turnOffServerAndVms(connection)
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/api/turnon", func(w http.ResponseWriter, r *http.Request) {
		go turnOnServerAndVms(connection)
		w.WriteHeader(http.StatusOK)
	})

	log.Info().Str("port", *port).Msg("starting server")
	err = http.ListenAndServe(":"+*port, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}

func manageEsxiServer(connection esxi.Connection) {
	online := connection.ServerReachable()
	log.Info().Bool("online", online).Msg("server online on manager start-up")

	for {
		if !online && schedule.IsInOperatingHours() {
			turnOnServerAndVms(connection)
			online = true

			log.Info().Msg("server started")
		}

		if online && !schedule.IsInOperatingHours() {
			turnOffServerAndVms(connection)
			online = false

			log.Info().Msg("server shutdown")
		}

		time.Sleep(10 * time.Second)
	}
}

func turnOnServerAndVms(connection esxi.Connection) error {
	log.Info().Msg("turning server on")

	err := connection.SendTurnOnCommand()
	if err != nil {
		log.Error().Err(err).Msg("failed to turn on server")
		return err
	}

	for i := 0; i < retryCount; i++ {
		if connection.ServerReachable() {
			log.Info().Msg("server online")
			break
		}

		if i == retryCount-1 {
			log.Warn().Int("retries", retryCount).Int("delay", retryDelay).Msg("server attempted to start, but isn't online")
		}

		log.Info().Int("delaySeconds", retryDelay).Msg("couldn't contact server, retrying...")

		time.Sleep(time.Duration(retryDelay) * time.Second)
	}

	connection.SetMaintainanceMode(false)
	connection.BootAllVMs()

	return nil
}

func turnOffServerAndVms(connection esxi.Connection) error {
	log.Info().Msg("turning server off")

	connection.ShutDownAllVMs()

	// TODO: Change this to monitor the state of VMs directly, and wait until they're all
	// shut down properly. This is just a temporary stopgap
	time.Sleep(1 * time.Minute)

	connection.SetMaintainanceMode(true)

	err := connection.SendTurnOffCommand()
	if err != nil {
		log.Error().Err(err).Msg("failed to turn off server")
		return err
	}

	for i := 0; i < retryCount; i++ {
		if !connection.ServerReachable() {
			log.Info().Msg("server offline")
			break
		}

		if i == retryCount-1 {
			log.Warn().Int("retries", retryCount).Int("delay", retryDelay).Msg("server attempted to shutdown, but isn't offline")
		}

		log.Info().Int("delay", retryDelay).Msg("server still online, retrying shutdown check...")
		time.Sleep(time.Duration(retryDelay) * time.Second)
	}

	return nil
}
