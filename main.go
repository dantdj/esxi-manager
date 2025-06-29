package main

import (
	"log"
	"os"
	"time"

	"github.com/dantdj/esxi-manager/internal/esxi"
	"github.com/dantdj/esxi-manager/internal/schedule"
	"github.com/joho/godotenv"
)

func main() {
	retryDelay := 30
	retryCount := 5

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("failed to load .env")
	}

	connection := esxi.Connection{
		Username:   os.Getenv("ESXI_USER"),
		Password:   os.Getenv("ESXI_PASS"),
		URL:        os.Getenv("ESXI_URL"),
		MACAddress: os.Getenv("ESXI_MAC"),
	}

	online := connection.ServerReachable()
	log.Printf("server online on manager start-up: %t", online)

	for {
		if !online && schedule.IsInOperatingHours() {
			log.Printf("turning server on")

			err := connection.SendTurnOnCommand()
			if err != nil {
				log.Printf("failed to turn on server: %s", err)
				// Try again later
				time.Sleep(10 * time.Second)
				continue
			}

			for {
				retries := 0
				if connection.ServerReachable() {
					log.Printf("server online")
					break
				}

				if retries >= retryCount {
					log.Printf("server attempted to start, but isn't online after %d seconds", retryDelay*retryCount)
				}

				log.Printf("couldn't contact server, retrying in %ds...\n", retryDelay)

				retries++
				time.Sleep(time.Duration(retryDelay) * time.Second)
			}

			online = true

			connection.SetMaintainanceMode(false)
			connection.BootAllVMs()

			log.Println("server started")
		}

		if online && !schedule.IsInOperatingHours() {
			log.Printf("turning server off")

			connection.ShutDownAllVMs()

			// TODO: Change this to monitor the state of VMs directly, and wait until they're all
			// shut down properly. This is just a temporary stopgap
			time.Sleep(2 * time.Minute)

			connection.SetMaintainanceMode(true)

			err := connection.SendTurnOffCommand()
			if err != nil {
				log.Printf("failed to turn off server: %s", err)
				// Try again
				continue
			}

			for {
				retries := 0
				if !connection.ServerReachable() {
					log.Printf("server offline")
					break
				}

				if retries >= retryCount {
					log.Printf("server attempted to shutdown, but isn't offline after %d seconds", retryDelay*retryCount)
				}

				retries++
				time.Sleep(time.Duration(retryDelay) * time.Second)
			}

			online = false
		}

		time.Sleep(10 * time.Second)
	}
}
