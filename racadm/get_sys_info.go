package racadm

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type GetSysInfo struct {
	CMCDateTime        time.Time
	PrimaryCMCLocation string
	PrimaryCMCVersion  string
	StandbyCMCVersion  string
	LastFirmwareUpdate time.Time
	HardwareVersion    string

	NICEnabled         bool
	MACAddress         net.HardwareAddr
	RegisterDNSCMCName bool
	DNSCMCName         string
	CurrentDNSDomain   string
	VLANID             int
	VLANPriority       int
	VLANEnabled        bool

	IPv4Enabled        bool
	CurrentIPAddress   net.IP
	CurrentIPGateway   net.IP
	CurrentIPNetmask   net.IPMask
	DHCPEnabled        bool
	CurrentDNSServer1  net.IP
	CurrentDNSServer2  net.IP
	DNSServersfromDHCP bool

	IPv6Enabled              bool
	AutoconfigurationEnabled bool
	LinkLocalAddress         net.IP
	CurrentIPv6Address1      net.IP
	CurrentIPv6Gateway       net.IP
	CurrentIPv6DNSServer1    net.IP
	CurrentIPv6DNSServer2    net.IP
	DNSServersfromDHCPv6     bool

	SystemModel            string
	SystemAssetTag         string
	ServiceTag             string
	ChassisName            string
	ChassisLocation        string
	ChassisMidplaneVersion string
	PowerStatus            string
	SystemID               string
}

func (c *Client) GetSysInfo() (*GetSysInfo, error) {
	var resp *GetSysInfo
	err := c.runCommand("racadm getsysinfo", func(r io.Reader) error {
		var err error
		if resp, err = parseGetSysInfo(r); err != nil {
			return fmt.Errorf("failed to parse output: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func parseGetSysInfo(r io.Reader) (*GetSysInfo, error) {
	var out GetSysInfo
	err := parseOutput(r, parseConfig{
		splitFn: func(in string) (string, []string, error) {
			txt := strings.TrimSpace(in)
			if txt == "" {
				return "", nil, errSkip
			}
			idx := strings.Index(txt, "=")
			if idx == -1 {
				return "", nil, errSkip
			}
			key := strings.TrimSpace(txt[:idx])
			val := strings.TrimSpace(txt[idx+1:])
			return key, []string{val}, nil
		},
		extractors: map[string]extract{
			"CMC Date/Time":        setTime(&out.CMCDateTime),
			"Primary CMC Location": setString(&out.PrimaryCMCLocation),
			"Primary CMC Version":  setString(&out.PrimaryCMCVersion),
			"Standby CMC Version":  setString(&out.StandbyCMCVersion),
			"Last Firmware Update": setTime(&out.LastFirmwareUpdate),
			"Hardware Version":     setString(&out.HardwareVersion),

			"NIC Enabled":           setBool(&out.NICEnabled),
			"MAC Address":           setMAC(&out.MACAddress),
			"Register DNS CMC Name": setBool(&out.RegisterDNSCMCName),
			"DNS CMC Name":          setString(&out.DNSCMCName),
			"Current DNS Domain":    setString(&out.CurrentDNSDomain),
			"VLAN ID":               setInt(&out.VLANID),
			"VLAN Priority":         setInt(&out.VLANPriority),
			"VLAN Enabled":          setBool(&out.VLANEnabled),

			"IPv4 Enabled":          setBool(&out.IPv4Enabled),
			"Current IP Address":    setIP(&out.CurrentIPAddress),
			"Current IP Gateway":    setIP(&out.CurrentIPGateway),
			"Current IP Netmask":    setIPMask(&out.CurrentIPNetmask),
			"DHCP Enabled":          setBool(&out.DHCPEnabled),
			"Current DNS Server 1":  setIP(&out.CurrentDNSServer1),
			"Current DNS Server 2":  setIP(&out.CurrentDNSServer2),
			"DNS Servers from DHCP": setBool(&out.DNSServersfromDHCP),

			"IPv6 Enabled":              setBool(&out.IPv6Enabled),
			"Autoconfiguration Enabled": setBool(&out.AutoconfigurationEnabled),
			"Link Local Address":        setIP(&out.LinkLocalAddress),
			"Current IPv6 Address 1":    setIP(&out.CurrentIPv6Address1),
			"Current IPv6 Gateway":      setIP(&out.CurrentIPv6Gateway),
			"Current IPv6 DNS Server 1": setIP(&out.CurrentIPv6DNSServer1),
			"Current IPv6 DNS Server 2": setIP(&out.CurrentIPv6DNSServer2),
			"DNS Servers from DHCPv6":   setBool(&out.DNSServersfromDHCPv6),

			"System Model":             setString(&out.SystemModel),
			"System AssetTag":          setString(&out.SystemAssetTag),
			"Service Tag":              setString(&out.ServiceTag),
			"Chassis Name":             setString(&out.ChassisName),
			"Chassis Location":         setString(&out.ChassisLocation),
			"Chassis Midplane Version": setString(&out.ChassisMidplaneVersion),
			"Power Status":             setString(&out.PowerStatus),
			"System ID":                setString(&out.SystemID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}
	return &out, nil
}
