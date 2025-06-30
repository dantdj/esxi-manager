# esxi-manager
Simple Go tool to facilitate managing power-on hours of an ESXi server.

Essentially a simple wrapper around sending Wake-on-LAN packets and SSH commands to manipulate the server as needed.

Supports:

* Turning the server on during operating hours, and off when outside of operating hours
* Turns on all VMs when booting, and turns off all VMs before shutting down

## Usage Notes
Currently need to set `PasswordAuthentication` in the `/etc/ssh/sshd_config` file on the ESXi host to `yes`. Hoping to fix this at a later date.

Set the following environment variables:

* ESXI_USER - the username of the user account on the ESXi server to use
* ESXI_PASS - the password of the user account on the ESXi server to use
* ESXI_URL - the URL of your ESXi server (used to SSH to your server)
* ESXI_MAC - the MAC address of your ESXi server (used to send the Wake-on-LAN packet)

The program supports loading these from a .env file in the same directory as the executable.

## Future plans

* Integrating the scheduling into the web interface, and make the scheduler and web interface usable at the same time