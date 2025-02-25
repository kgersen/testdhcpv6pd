# testdhcpv6pd

Sends a DHCPv6-PD solicit message and displays the response (no request is done)

## usage

````text
testdhcpv6pd [options] [interface name] or [interface index]

Available options:
  -a string
        anonymize ip addresses (format = list word indexes to show) (default "12345678")
  -dll string
        specify type 3 DUID-LL using the provided mac address ( : or - separated digits)
  -dllt string
        specify type 1 DUID-LLT using the provided mac address ( : or - separated digits)
  -dlltt uint
        specify the Time field for DUID-LLT
  -duu string
        specify type 4 DUID-UUID (format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
  -p value
        ask for a specific prefix and/or length (repeatable, default is one prefix of ::/64)
  -s    dont print debug messages
  -test
        dry-run only,  print the solicit paquet, nothing is send on the network
  -v    display version````

Without argument, `testdhcpv6pd` will display the available interfaces

Use `-s` to suppress packet debugging 

Use `-a format` to anonymize the prefix where format is a list of indexes. Each index, from 1 to 8, is the part number of the address to keep. For instance `-a "18"` only keeps the first and last part of the address.

Use `-p ::/60` to request a /60 prefix or even `-p 2a01:xxx:xxxx:xxxx::/64,` to request a specific prefix. can be repeated. `iaid` used 1, 2, etc

Other options allow to change the DUID.

## notes

Not tested on *bsd, plan9

Linux systems require *at least* `cap_net_bind_service` capability to bind to port 546 (see `man 7 capabilities`) or just use `sudo`

Darwin/MacOS: use `sudo`
