package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/dantdj/esxi-manager/internal/esxi"
	"github.com/dantdj/esxi-manager/internal/schedule"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var retryDelay int = 30
var retryCount int = 5

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load .env")
	}

	connection := esxi.Connection{
		Username:   os.Getenv("ESXI_USER"),
		Password:   os.Getenv("ESXI_PASS"),
		URL:        os.Getenv("ESXI_URL"),
		MACAddress: os.Getenv("ESXI_MAC"),
	}

	manage := flag.Bool("manage", true, "Enable or disable the ESXi power-on/power-off schedule")
	flag.Parse()

	if *manage {
		go manageEsxiServer(connection)
	}

	fs := http.FileServer(http.Dir("./web/dist"))
	http.Handle("/", fs)
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

	log.Info().Msg("starting server on :8080")
	http.ListenAndServe(":8080", nil)
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
