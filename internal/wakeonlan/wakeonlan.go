package wakeonlan

import (
	"errors"
	"net"

	"github.com/mdlayher/wol"
)

// Send a Wake-on-LAN packet to the given MAC address.
// Assumes a broadcast address of 255.255.255.255:9
// Returns any errors that occur during the process.
func SendWolPacket(mac string) error {
	wolC, err := wol.NewClient()
	if err != nil {
		return err
	}

	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return errors.New("Failed to parse MAC: " + err.Error())
	}

	if err := wolC.Wake("255.255.255.255:9", hwAddr); err != nil {
		return err
	}

	return nil
}
