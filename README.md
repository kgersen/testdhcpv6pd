## testdhcpv6pd

Sends a DHCPv6-PD solicit message and displays the response (no request is done)

### usage:

`testdhcpv6pd [-l prefix-length] [-s] [interface name or index]`

Without argument, `testdhcpv6pd` will display the available interfaces

Use `-s` to suppress packet debugging 

### notes
Not tested on *bsd, darwin, plan9

On Linux systems requires *at least* `cap_net_bind_service` capability to bind to port 546 (see `man 7 capabilities`)