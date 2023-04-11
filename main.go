package main

import (
	"errors"
	"flag"
	"log"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/client6"
	"github.com/insomniacslk/dhcp/iana"
)

var (
	iface = flag.String("i", "eth0", "Interface to configure via DHCPv6")
)

func main() {
	flag.Parse()
	log.Printf("Starting DHCPv6 client on interface %s", *iface)

	// NewClient sets up a new DHCPv6 client with default values
	// for read and write timeouts, for destination address and listening
	// address
	client := client6.NewClient()

	// Exchange runs a Solicit-Advertise-Request-Reply transaction on the
	// specified network interface, and returns a list of DHCPv6 packets
	// (a "conversation") and an error if any. Notice that Exchange may
	// return a non-empty packet list even if there is an error. This is
	// intended, because the transaction may fail at any point, and we
	// still want to know what packets were exchanged until then.
	// A default Solicit packet will be used during the "conversation",
	// which can be manipulated by using modifiers.
	//conversation, err := client.Exchange(*iface)

	sol, adv, err := Solicit(client, *iface, dhcpv6.WithIAPD(
		[4]byte{1, 0, 0, 0},
		&dhcpv6.OptIAPrefix{
			PreferredLifetime: 0,
			ValidLifetime:     0,
			Prefix: &net.IPNet{
				Mask: net.CIDRMask(64, 128),
				IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			Options: dhcpv6.PrefixOptions{Options: dhcpv6.Options{}},
		},
	))

	// Summary() prints a verbose representation of the exchanged packets.
	log.Print(sol.Summary())
	if adv != nil {
		log.Print(adv.Summary())
	}
	// error handling is done *after* printing, so we still print the
	// exchanged packets if any, as explained above.
	if err != nil {
		log.Fatal(err)
	}
}

// NewSolicit creates a new SOLICIT message, using the given hardware address to
// derive the IAID in the IA_NA option.
func NewSolicit(hwaddr net.HardwareAddr, modifiers ...dhcpv6.Modifier) (*dhcpv6.Message, error) {
	duid := &dhcpv6.DUIDLLT{
		HWType:        iana.HWTypeEthernet,
		Time:          dhcpv6.GetTime(),
		LinkLayerAddr: hwaddr,
	}
	m, err := dhcpv6.NewMessage()
	if err != nil {
		return nil, err
	}
	m.MessageType = dhcpv6.MessageTypeSolicit
	m.AddOption(dhcpv6.OptClientID(duid))
	m.AddOption(dhcpv6.OptRequestedOption(
		dhcpv6.OptionDNSRecursiveNameServer,
		dhcpv6.OptionDomainSearchList,
	))
	m.AddOption(dhcpv6.OptElapsedTime(0))
	if len(hwaddr) < 4 {
		return nil, errors.New("short hardware addrss: less than 4 bytes")
	}
	for _, mod := range modifiers {
		mod(m)
	}
	return m, nil
}

// Solicit sends a Solicit, returns the Solicit, an Advertise (if not nil), and
// an error if any. The modifiers will be applied to the Solicit before sending
// it, see modifiers.go
func Solicit(client *client6.Client, ifname string, modifiers ...dhcpv6.Modifier) (dhcpv6.DHCPv6, dhcpv6.DHCPv6, error) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, nil, err
	}
	solicit, err := NewSolicit(iface.HardwareAddr,modifiers...)
	if err != nil {
		return nil, nil, err
	}
	advertise, err := client.SendReceive(ifname, solicit, dhcpv6.MessageTypeNone)
	return solicit, advertise, err
}
