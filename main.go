package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/nclient6"
	"github.com/insomniacslk/dhcp/iana"
)

var interfaces []net.Interface

func init() {
	var err error
	interfaces, err = net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
}

func displayInterfaces() {
	fmt.Println("available interface - name (index):")
	for _, v := range interfaces {
		fmt.Printf("%s (%d)\n", v.Name, v.Index)
	}
}
func parseInterface(name string) (*net.Interface, error) {
	//try name
	i, err := net.InterfaceByName(name)
	if err == nil {
		return i, nil
	}
	//try index
	if n, err := strconv.Atoi(name); err != nil {
		i, err := net.InterfaceByIndex(n)
		if err != nil {
			return nil, fmt.Errorf("interface index not found")
		}
		return i, nil
	}
	return nil, fmt.Errorf("interface not found")
}
func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Printf("Usage: %s [interface name] or [interface index]\n", os.Args[0])
		displayInterfaces()
		os.Exit(0)
	}

	iface, err := parseInterface(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Starting DHCPv6 client on interface %s", iface.Name)

	// NewClient sets up a new DHCPv6 client with default values
	// for read and write timeouts, for destination address and listening
	// address
	client, err := nclient6.New(iface.Name,
		nclient6.WithTimeout(2*time.Second),
		nclient6.WithRetry(1),
		nclient6.WithDebugLogger())

	if err != nil {
		log.Fatal(err)
	}

	// Exchange runs a Solicit-Advertise-Request-Reply transaction on the
	// specified network interface, and returns a list of DHCPv6 packets
	// (a "conversation") and an error if any. Notice that Exchange may
	// return a non-empty packet list even if there is an error. This is
	// intended, because the transaction may fail at any point, and we
	// still want to know what packets were exchanged until then.
	// A default Solicit packet will be used during the "conversation",
	// which can be manipulated by using modifiers.
	//conversation, err := client.Exchange(*iface)

	adv, err := Solicit(context.Background(), client, dhcpv6.WithIAPD(
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
	// l := len(hwaddr)
	// var iaid [4]byte
	// copy(iaid[:], hwaddr[l-4:l])
	//modifiers = append([]dhcpv6.Modifier{dhcpv6.WithIAID(iaid)}, modifiers...)
	// Apply modifiers
	for _, mod := range modifiers {
		mod(m)
	}
	return m, nil
}

// Solicit sends a solicitation message and returns the first valid
// advertisement received.
func Solicit(ctx context.Context, c *nclient6.Client, modifiers ...dhcpv6.Modifier) (*dhcpv6.Message, error) {
	solicit, err := NewSolicit(c.InterfaceAddr(), modifiers...)
	if err != nil {
		return nil, err
	}
	msg, err := c.SendAndRead(ctx, c.RemoteAddr(), solicit, nclient6.IsMessageType(dhcpv6.MessageTypeAdvertise))
	if err != nil {
		return nil, err
	}
	return msg, nil
}
