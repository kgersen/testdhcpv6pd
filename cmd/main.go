package main

import (
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/iana"
	dhcp6c "github.com/nspeed-app/testdhcpv6pd"

	"nspeed.app/nspeed/utils"
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

type myLogger struct {
	*log.Logger
	Debug     bool
	Anonymize string
}

func NewMyLogger() myLogger {
	return myLogger{Logger: log.New(os.Stderr, "[dhcpv6] ", log.LstdFlags)}
}

func (e *myLogger) Printf(format string, v ...interface{}) {
	if e.Debug {
		e.Logger.Printf(format, v...)
	}
}
func (e *myLogger) PrintMessage(prefix string, message *dhcpv6.Message) {
	if e.Debug {
		e.Printf("%s: %s", prefix, message.Summary())
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

// prefixesFlag implement flag.Value interface
type prefixesFlag []string

// String() for flag.Value interface
func (i *prefixesFlag) String() string {
	return fmt.Sprintf("%v", *i)
}

// Set() for flag.Value interface
func (i *prefixesFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var optPrefixes prefixesFlag

var (
	optNoDebug   = flag.Bool("s", false, "dont print debug messages")
	optVersion   = flag.Bool("v", false, "display version")
	optAnonymize = flag.String("a", utils.FormatV6Full, "anonymize ip addresses (format = list word indexes to show)")
	optDUID1     = flag.String("dllt", "", "specify type 1 DUID-LLT using the provided mac address ( : or - separated digits)")
	optDUID1T    = flag.Uint("dlltt", 0, "specify the Time field for DUID-LLT")
	optDUID3     = flag.String("dll", "", "specify type 3 DUID-LL using the provided mac address ( : or - separated digits)")
	optDUID4     = flag.String("duu", "", "specify type 4 DUID-UUID (format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)")
	optDryRun    = flag.Bool("test", false, "dry-run only,  print the solicit paquet, nothing is send on the network")
)

func main() {

	flag.Var(&optPrefixes, "p", "ask for a specific prefix and/or length (repeatable, default is one prefix of ::/64)")
	flag.Parse()

	if *optVersion {
		fmt.Println("version", version)
		os.Exit(0)
	}
	if len(flag.Args()) != 1 {
		fmt.Printf("Usage: %s [options] [interface name] or [interface index]\n", os.Args[0])
		displayInterfaces()
		fmt.Println("\nAvailable options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// parse prefix(es)
	if optPrefixes == nil {
		optPrefixes = append(optPrefixes, "::/64")
	}

	var prefixes []netip.Prefix
	for _, p := range optPrefixes {
		prefix, err := netip.ParsePrefix(p)
		if err != nil {
			log.Fatal("bad prefix", p, err)
		}
		prefixes = append(prefixes, prefix)
	}

	// parse interface
	iface, err := parseInterface(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	if !*optNoDebug {
		log.Printf("Sending a DHCPv6-PD Solicit on interface %s", iface.Name)
	}

	logger := NewMyLogger()
	logger.Debug = !*optNoDebug
	logger.Anonymize = *optAnonymize
	var client *dhcp6c.Client
	client, err = dhcp6c.New(iface.Name,
		dhcp6c.WithTimeout(2*time.Second),
		dhcp6c.WithRetry(1),
		dhcp6c.WithLogger(&logger))

	if err != nil {
		log.Fatal(err)
	}

	// MacOs/darwin needs Zone set to same interface or 'no route to host' error
	// since this doesn't bother other OSes  , we generalize this
	if true { // runtime.GOOS == "darwin" {
		baddr := dhcp6c.AllDHCPRelayAgentsAndServers
		baddr.Zone = iface.Name
		dhcp6c.WithBroadcastAddr(baddr)(client)
	}
	// build solicit options
	var modifiers []dhcpv6.Modifier
	for i, prefix := range prefixes {
		iaid := [4]byte{}
		binary.BigEndian.PutUint32(iaid[:], uint32(i+1))
		modifiers = append(modifiers, dhcp6c.WithIAPD(
			iaid,
			&dhcpv6.OptIAPrefix{
				PreferredLifetime: 0,
				ValidLifetime:     0,
				Prefix: &net.IPNet{
					Mask: net.CIDRMask(prefix.Bits(), 128),
					IP:   prefix.Addr().AsSlice(),
				},
				Options: dhcpv6.PrefixOptions{Options: dhcpv6.Options{}},
			}))
	}

	/* https://datatracker.ietf.org/doc/html/rfc8415#section-11

	   A DUID consists of a 2-octet type code represented in network byte
	   order, followed by a variable number of octets that make up the
	   actual identifier.  The length of the DUID (not including the type
	   code) is at least 1 octet and at most 128 octets.  The following
	   types are currently defined:

	      +------+------------------------------------------------------+
	      | Type | Description                                          |
	      +------+------------------------------------------------------+
	      | 1    | Link-layer address plus time                         |
	      | 2    | Vendor-assigned unique ID based on Enterprise Number |
	      | 3    | Link-layer address                                   |
	      | 4    | Universally Unique Identifier (UUID) [RFC6355]       |
	      +------+------------------------------------------------------+

	                            Table 2: DUID Types

	   Formats for the variable field of the DUID for the first three of the
	   above types are shown below.  The fourth type, DUID-UUID [RFC6355],
	   can be used in situations where there is a UUID stored in a device's
	   firmware settings.
	*/

	// default duid is type 1
	hmac := client.InterfaceAddr()
	t := dhcpv6.GetTime()
	DUIDset := false
	if *optDUID1 != "" {
		DUIDset = true
		hmac, err = net.ParseMAC(*optDUID1)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *optDUID1T != 0 {
		t = uint32(*optDUID1T)
	}
	var duid dhcpv6.DUID = &dhcpv6.DUIDLLT{
		HWType:        iana.HWTypeEthernet,
		Time:          t,
		LinkLayerAddr: hmac,
	}

	// type 2 ? (todo?)

	// type 3
	if *optDUID3 != "" {
		if DUIDset {
			log.Fatal("DUID already specified")
		}
		DUIDset = true
		mac, err := net.ParseMAC(*optDUID3)
		if err != nil {
			log.Fatal(err)
		}
		duid = &dhcpv6.DUIDLL{
			HWType:        iana.HWTypeEthernet,
			LinkLayerAddr: mac,
		}

	}
	// type 4
	if *optDUID4 != "" {
		if DUIDset {
			log.Fatal("DUID already specified")
		}
		DUIDset = true
		uuid, err := uuid.Parse(*optDUID4)
		if err != nil {
			log.Fatal(err)
		}
		duid = &dhcpv6.DUIDUUID{
			UUID: uuid,
		}
	}
	// disabled for now, as it's not conform to the rfc
	// if *optCID != "" {
	// 	mac, err := net.ParseMAC(*optCID)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	modifiers = append(modifiers, dhcpv6.WithClientID(&dhcpv6.DUIDLLT{
	// 		HWType:        iana.HWTypeEthernet,
	// 		Time:          dhcpv6.GetTime(),
	// 		LinkLayerAddr: mac,
	// 	}))
	// }

	adv, err := Solicit(context.Background(), *optDryRun, duid, client, modifiers...)

	// Summary() prints a verbose representation of the exchanged packets.
	if adv != nil {
		if adv.MessageType != dhcpv6.MessageTypeAdvertise {
			log.Fatal("unexcepted message type")
		}
		opts := adv.GetOption(dhcpv6.OptionIAPD)
		if opts == nil {
			log.Fatal("no IAPD found")
		}
		for _, opt := range opts {
			iapd := dhcpv6.OptIAPD{}
			if err := iapd.FromBytes(opt.ToBytes()); err != nil {
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
	}
	// error handling is done *after* printing, so we still print the
	// exchanged packets if any, as explained above.
	if err != nil {
		log.Fatal(err)
	}
}

// NewSolicit creates a new SOLICIT message with given duid
// derive the IAID in the IA_NA option.
func NewSolicit(duid dhcpv6.DUID, modifiers ...dhcpv6.Modifier) (*dhcpv6.Message, error) {
	if duid == nil {
		return nil, errors.New("no duid")
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
	for _, mod := range modifiers {
		mod(m)
	}
	return m, nil
}

// Solicit sends a solicitation message and returns the first valid
// advertisement received.
func Solicit(ctx context.Context, dryRun bool, duid dhcpv6.DUID, c *dhcp6c.Client, modifiers ...dhcpv6.Modifier) (*dhcpv6.Message, error) {
	solicit, err := NewSolicit(duid, modifiers...)
	if err != nil {
		return nil, err
	}
	if dryRun {
		c.PrintMessage("will send:", solicit)
		return nil, nil
	}
	msg, err := c.SendAndRead(ctx, c.RemoteAddr(), solicit, dhcp6c.IsMessageType(dhcpv6.MessageTypeAdvertise))
	if err != nil {
		return nil, err
	}
	return msg, nil
}
