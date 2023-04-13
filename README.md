## testdhcpv6pd

Sends a DHCPv6-PD solicit message and displays the response (no request is done)

### usage:

````
testdhcpv6pd [options] [interface name or index]

    available options:
    -a string
            anonymize ip addresses (format = list nibble indexes to show) (default "12345678")
    -l int
            prefix length (default 64)
    -s	dont print debug messages
    -v	display version
````
Without argument, `testdhcpv6pd` will display the available interfaces

Use `-s` to suppress packet debugging 

Use `-a format` to anonymize the prefix where format is a list of indexes. Each index, from 1 to 8, is the part number of the address to keep. For instance `-a "18"` only keeps the first and last part of the address.

### notes
Not tested on *bsd, plan9

Linux systems require *at least* `cap_net_bind_service` capability to bind to port 546 (see `man 7 capabilities`) or just use `sudo`
Darwin/MacOS: use `sudo`