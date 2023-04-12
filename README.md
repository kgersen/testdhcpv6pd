## testdhcpv6pd

sends a DHCPv6-PD solicit message and display the response

### usage:

`testdhcpv6pd [-l prefix-length] [-s] [interface name or index]`

without argument, `testdhcpv6pd` will display the available interfaces

use `-s` to suppress packet debug 

### notes
not tested on *bsd, darwin