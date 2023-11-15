module github.com/kgersen/testdhcpv6pd

go 1.20

require (
	github.com/insomniacslk/dhcp v0.0.0-20230407062729-974c6f05fe16
	github.com/nspeed-app/nspeed v0.0.10
)

require (
	github.com/josharian/native v1.1.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.14 // indirect
	github.com/u-root/uio v0.0.0-20230220225925-ffce2a382923 // indirect
	golang.org/x/sys v0.5.0 // indirect
)

replace github.com/insomniacslk/dhcp => ../go-dhcp
