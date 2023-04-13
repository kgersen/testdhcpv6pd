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

	"github.com/nspeed-app/nspeed/utils"
)

// version will be filled by goreleaser
var version string

var interfaces []net.Interface

func init() {
	var err error
	interfaces, err = net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
}

func displayInterfaces() {
	fmt.Printf("available interface - name (index):\n\n")
	for _, v := range interfaces {
		fmt.Printf("  %s (%d)\n", v.Name, v.Index)
	}
}
func parseInterface(name string) (*net.Interface, error) {
	//try name
	i, err := net.InterfaceByName(name)
	if err == nil {
		return i, nil
	}
	//try index
	if n, err := strconv.Atoi(name); err == nil {
		i, err := net.InterfaceByIndex(n)
		if err != nil {
			return nil, fmt.Errorf("interface index not found")
		}
		return i, nil
	}
	return nil, fmt.Errorf("interface not found")
}

var (
	optPrefixLength = flag.Int("l", 64, "prefix length")
	optNoDebug      = flag.Bool("s", false, "dont print debug messages")
	optVersion      = flag.Bool("v", false, "display version")
	optAnonymize    = flag.String("a", utils.FormatV6Full, "anonymize ip addresses (format = list word indexes to show)")
)

func main() {

	flag.Parse()

	if *optVersion {
		fmt.Println("version", version)
		os.Exit(0)
	}
	if *optPrefixLength < 1 || *optPrefixLength > 128 {
		log.Fatal("prefix length")
	}
	if len(flag.Args()) != 1 {
		fmt.Printf("Usage: %s [options] [interface name] or [interface index]\n", os.Args[0])
		displayInterfaces()
		fmt.Println("\nAvailable options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	iface, err := parseInterface(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	if !*optNoDebug {
		log.Printf("Sending a DHCPv6-PD Solicit on interface %s", iface.Name)
	}

	var client *nclient6.Client
	client, err = nclient6.New(iface.Name,
		nclient6.WithTimeout(2*time.Second),
		nclient6.WithRetry(1))

	if err != nil {
		log.Fatal(err)
	}

	if !*optNoDebug {
		nclient6.WithDebugLogger()(client)
	}

	// MacOs/darwin needs Zone set to same interface or 'no route to host' error
	// since this doesn't bother other OSes  , we generalize this
	if true { // runtime.GOOS == "darwin" {
		baddr := nclient6.AllDHCPRelayAgentsAndServers
		baddr.Zone = iface.Name
		nclient6.WithBroadcastAddr(baddr)(client)
	}
	// send a Solicit with IAPD, no IAID
	adv, err := Solicit(context.Background(), client, dhcpv6.WithIAPD(
		[4]byte{1, 0, 0, 0},
		&dhcpv6.OptIAPrefix{
			PreferredLifetime: 0,
			ValidLifetime:     0,
			Prefix: &net.IPNet{
				Mask: net.CIDRMask(*optPrefixLength, 128),
				IP:   net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			Options: dhcpv6.PrefixOptions{Options: dhcpv6.Options{}},
		},
	))

	// Summary() prints a verbose representation of the exchanged packets.
	if adv != nil {
		if adv.MessageType != dhcpv6.MessageTypeAdvertise {
			log.Fatal("unexcepted message type")
		}
		IAPDOption := adv.GetOneOption(dhcpv6.OptionIAPD)
		if IAPDOption == nil {
			log.Fatal("no IAPD found")
		}
		iapd := dhcpv6.OptIAPD{}
		if err := iapd.FromBytes(IAPDOption.ToBytes()); err != nil {
			log.Fatal("cant parse iadp")
		}
		prefixes := iapd.Options.Prefixes()
		if prefixes == nil {
			log.Fatal("no prefix found")
		}
		for _, p := range prefixes {
			log.Printf("got a prefix = %s (pttl=%s,vttl=%s)\n", utils.AnonymizeIPNet(p.Prefix, utils.FormatV4First, *optAnonymize), p.PreferredLifetime, p.ValidLifetime)
		}
	}
	// error handling is done *after* printing, so we still print the
	// exchanged packets if any, as explained above.
	if err != nil {
		log.Fatal(err)
	}
}

// NewSolicit creates a new SOLICIT message, using the given hardware address to
// derive the IAID in the IA_NA option.
// same as nclient6/NewSolicit but without IAID
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
// same as nclient6.Solicit but using NewSolicit above (no IAID)
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
