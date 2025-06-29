# esxi-manager
Simple Go tool to facilitate managing power-on hours of an ESXi server.

Supports turning the server on during operating hours, and off when outside of operating hours.

## Notes
Currently need to set `PasswordAuthentication` in the `/etc/ssh/sshd_config` file on the ESXi host to `yes`. Hoping to fix this at a later date.