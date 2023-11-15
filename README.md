## testdhcpv6pd

Sends a DHCPv6-PD solicit message and displays the response (no request is done)

### usage:

````
testdhcpv6pd [options] [interface name or index]

  -a string
    	anonymize ip addresses (format = list word indexes to show) (default "12345678")
  -cid string
    	specify client ID (DUID-LLT) using a mac address ( : or - separated digits)
  -cll string
    	specify client layer address using a mac address ( : or - separated digits)
  -p string
    	ask for a specific prefix and/or length (default "::/64")
  -s	dont print debug messages
  -v	display version

````
Without argument, `testdhcpv6pd` will display the available interfaces

Use `-s` to suppress packet debugging 

Use `-a format` to anonymize the prefix where format is a list of indexes. Each index, from 1 to 8, is the part number of the address to keep. For instance `-a "18"` only keeps the first and last part of the address.

Use `-p ::/60` to request a /60 prefix or even `-p 2a01:xxx:xxxx:xxxx::/64,` to request a specific prefix

Use `-cid aa:bb:cc:dd:ee:ff` to change the DUID (LLT format, RFC 8415 Section 11.2). Format is a MAC address, : or - separators.

Use `-cll aa:bb:cc:dd:ee:ff` to add a link layer address option (RFC 6939). Format is a MAC address, : or - separators.

### notes
Not tested on *bsd, plan9

Linux systems require *at least* `cap_net_bind_service` capability to bind to port 546 (see `man 7 capabilities`) or just use `sudo`

Darwin/MacOS: use `sudo`