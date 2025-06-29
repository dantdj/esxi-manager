package main

import (
	"os"
	"time"

	"github.com/dantdj/esxi-manager/internal/esxi"
	"github.com/dantdj/esxi-manager/internal/schedule"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	retryDelay := 30
	retryCount := 5

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

	online := connection.ServerReachable()
	log.Info().Bool("online", online).Msg("server online on manager start-up")

	for {
		if !online && schedule.IsInOperatingHours() {
			log.Info().Msg("turning server on")

			err := connection.SendTurnOnCommand()
			if err != nil {
				log.Error().Err(err).Msg("failed to turn on server")
				// Try again later
				time.Sleep(10 * time.Second)
				continue
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

			online = true

			connection.SetMaintainanceMode(false)
			connection.BootAllVMs()

			log.Info().Msg("server started")
		}

		if online && !schedule.IsInOperatingHours() {
			log.Info().Msg("turning server off")

			connection.ShutDownAllVMs()

			// TODO: Change this to monitor the state of VMs directly, and wait until they're all
			// shut down properly. This is just a temporary stopgap
			time.Sleep(2 * time.Minute)

			connection.SetMaintainanceMode(true)

			err := connection.SendTurnOffCommand()
			if err != nil {
				log.Error().Err(err).Msg("failed to turn off server")
				// Try again
				continue
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

			online = false
		}

		time.Sleep(10 * time.Second)
	}
}
