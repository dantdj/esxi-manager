package esxi

import (
	"fmt"
	"log"
	"time"

	"github.com/dantdj/esxi-manager/internal/wakeonlan"
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
		log.Printf("failed to send Wake-on-LAN packet: %s", err)
		return err
	}

	return nil
}

// Sends a command to turn off the current server
func (ec *Connection) SendTurnOffCommand() error {
	err := ec.sendSSHCommand("esxcli system shutdown poweroff --reason 'routine shutdown'")

	if err != nil {
		log.Printf("failed sending poweroff command: %s", err)
		return err
	}

	return nil
}

// Sets maintainance mode to the value provided.
func (ec *Connection) SetMaintainanceMode(value bool) error {
	log.Printf("setting maintainance mode to %t", value)
	command := fmt.Sprintf("esxcli system maintenanceMode set --enable %t", value)
	err := ec.sendSSHCommand(command)
	if err != nil {
		log.Printf("failed to set maintenance mode to %t: %s", value, err)
		return err
	}

	return nil
}

// Sends commands to boot the VMs with the provided IDs
func (ec *Connection) BootVMs(vmIds ...string) error {
	for _, vmId := range vmIds {
		err := ec.sendSSHCommand(fmt.Sprintf("esxcli vm process start --type=%s", vmId))
		if err != nil {
			log.Printf("failed to boot VM %s: %s", vmId, err)
			return err
		}
	}

	return nil
}

// Sends command to boot all VMs on the server
func (ec *Connection) BootAllVMs() error {
	log.Printf("attempting to boot all VMs")
	return ec.sendSSHCommand("for vmid in $(vim-cmd vmsvc/getallvms | awk 'NR>1{print $1}'); do vim-cmd vmsvc/power.on $vmid; done")
}

// Sends command to shut down all VMs on the server
func (ec *Connection) ShutDownAllVMs() error {
	log.Printf("attempting to shut down all VMs")
	return ec.sendSSHCommand("for vmid in $(vim-cmd vmsvc/getallvms | awk 'NR>1{print $1}'); do vim-cmd vmsvc/power.off $vmid; done")
}

// Send a generic SSH command to the current ESXi server
func (ec *Connection) sendSSHCommand(command string) error {
	log.Printf("sending command to server: %s", command)
	client, err := simplessh.ConnectWithPasswordTimeout(ec.URL, ec.Username, ec.Password, 5*time.Second)
	if err != nil {
		return err
	}
	defer client.Close()

	output, err := client.Exec(command)
	if err != nil {
		return err
	}

	log.Printf("received message from server: %s", string(output))

	return nil
}

// Returns whether or not the current ESXi server is reachable
func (ec *Connection) ServerReachable() bool {
	err := ec.sendSSHCommand("esxcli --version")
	if err != nil {

		log.Printf("error determining if server was reachable: %s", err)
	}
	// If we got an error back, then the server isn't reachable by default
	return err == nil
}
