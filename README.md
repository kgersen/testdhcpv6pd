## testdhcpv6pd

sends a DHCPv6-PD solicit message and display the response

### usage:

`testdhcpv6pd [-l prefix-length] [-s] [interface name or index]`

without argument, `testdhcpv6pd` will display the available interfaces

use `-s` to suppress packet debug 

### notes
Not tested on *bsd, darwin, plan9

On Linux systems requires *at least* `cap_net_bind_service` capability to bind to port 546 (see `man 7 capabilities`)