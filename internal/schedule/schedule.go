package schedule

import (
	"time"

	"github.com/rs/zerolog/log"
)

// Returns whether or not the current time is within the operating hours
// for the server.
func IsInOperatingHours() bool {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		log.Error().Err(err).Msg("error loading location")
		return false
	}
	currentTime := time.Now().In(loc)

	startOfHours := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 17, 0, 0, 0, currentTime.Location())
	endOfHours := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 20, 0, 0, 0, currentTime.Location())

	if currentTime.After(startOfHours) && currentTime.Before(endOfHours) {
		return true
	}

	return false
}
