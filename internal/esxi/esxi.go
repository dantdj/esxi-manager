package esxi

import (
	"fmt"
	"time"

	"github.com/dantdj/esxi-manager/internal/wakeonlan"
	"github.com/rs/zerolog/log"
	"github.com/sfreiberg/simplessh"
)

type Connection struct {
	URL        string
	MACAddress string
	Username   string
	Password   string
}

func New(url, username, password, mac string) Connection {
	return Connection{
		URL:        url,
		Username:   username,
		Password:   password,
		MACAddress: mac,
	}
}

// Sends a command to turn on the current server
func (ec *Connection) SendTurnOnCommand() error {
	err := wakeonlan.SendWolPacket(ec.MACAddress)
	if err != nil {
		log.Error().Err(err).Msg("failed to send Wake-on-LAN packet")
		return err
	}

	return nil
}

// Sends a command to turn off the current server
func (ec *Connection) SendTurnOffCommand() error {
	err := ec.sendSSHCommand("esxcli system shutdown poweroff --reason 'routine shutdown'")

	if err != nil {
		log.Error().Err(err).Msg("failed sending poweroff command")
		return err
	}

	return nil
}

// Sets maintainance mode to the value provided.
func (ec *Connection) SetMaintainanceMode(value bool) error {
	log.Info().Bool("value", value).Msg("setting maintainance mode")
	command := fmt.Sprintf("esxcli system maintenanceMode set --enable %t", value)
	err := ec.sendSSHCommand(command)
	if err != nil {
		log.Error().Err(err).Bool("value", value).Msg("failed to set maintenance mode")
		return err
	}

	return nil
}

// Sends commands to boot the VMs with the provided IDs
func (ec *Connection) BootVMs(vmIds ...string) error {
	for _, vmId := range vmIds {
		err := ec.sendSSHCommand(fmt.Sprintf("esxcli vm process start --type=%s", vmId))
		if err != nil {
			log.Error().Err(err).Str("vmId", vmId).Msg("failed to boot VM")
			return err
		}
	}

	return nil
}

// Sends command to boot all VMs on the server
func (ec *Connection) BootAllVMs() error {
	log.Info().Msg("attempting to boot all VMs")
	return ec.sendSSHCommand("for vmid in $(vim-cmd vmsvc/getallvms | awk 'NR>1{print $1}'); do vim-cmd vmsvc/power.on $vmid; done")
}

// Sends command to shut down all VMs on the server
func (ec *Connection) ShutDownAllVMs() error {
	log.Info().Msg("attempting to shut down all VMs")
	return ec.sendSSHCommand("for vmid in $(vim-cmd vmsvc/getallvms | awk 'NR>1{print $1}'); do vim-cmd vmsvc/power.off $vmid; done")
}

// Send a generic SSH command to the current ESXi server
func (ec *Connection) sendSSHCommand(command string) error {
	log.Info().Str("command", command).Msg("sending command to server")
	client, err := simplessh.ConnectWithPasswordTimeout(ec.URL, ec.Username, ec.Password, 5*time.Second)
	if err != nil {
		return err
	}
	defer client.Close()

	output, err := client.Exec(command)
	if err != nil {
		return err
	}

	log.Info().Str("message", string(output)).Msg("received message from server")

	return nil
}

// Returns whether or not the current ESXi server is reachable
func (ec *Connection) ServerReachable() bool {
	err := ec.sendSSHCommand("esxcli --version")
	if err != nil {

		log.Error().Err(err).Msg("error determining if server was reachable")
	}
	// If we got an error back, then the server isn't reachable by default
	return err == nil
}
