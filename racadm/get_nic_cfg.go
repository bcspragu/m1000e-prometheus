package racadm

import (
	"fmt"
	"io"
	"net"
	"strings"
)

type GetNICConfig struct {
	LOMModelName             string     // Embedded LOM
	LOMFabricType            string     // Gigabit Ethernet
	IPv4Enabled              bool       // 1
	DHCPEnabled              bool       // 0
	IPAddress                net.IP     // 192.168.2.16
	SubnetMask               net.IPMask // 255.255.255.0
	Gateway                  net.IP     // 192.168.2.1
	IPv6Enabled              bool       // 0
	AutoconfigurationEnabled bool       // 0
	// LinkLocalAddress         net.IP     //
	IPv6Gateway  net.IP // ::
	VLANEnable   bool   // 0
	VLANID       int    // 1
	VLANpriority int    // 0
}

func (c *Client) GetNICConfig(slotNum int) (*GetNICConfig, error) {
	var resp *GetNICConfig
	err := c.runCommand(fmt.Sprintf("racadm getniccfg -m server-%d", slotNum), func(r io.Reader) error {
		var err error
		if resp, err = parseGetNICConfig(r); err != nil {
			return fmt.Errorf("failed to parse output: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func parseGetNICConfig(r io.Reader) (*GetNICConfig, error) {
	var out GetNICConfig
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
			"LOM Model Name":            setString(&out.LOMModelName),
			"LOM Fabric Type":           setString(&out.LOMFabricType),
			"IPv4 Enabled":              setBool(&out.IPv4Enabled),
			"DHCP Enabled":              setBool(&out.DHCPEnabled),
			"IP Address":                setIP(&out.IPAddress),
			"Subnet Mask":               setIPMask(&out.SubnetMask),
			"Gateway":                   setIP(&out.Gateway),
			"IPv6 Enabled":              setBool(&out.IPv6Enabled),
			"Autoconfiguration Enabled": setBool(&out.AutoconfigurationEnabled),
			// "Link local Address":        setIP(&out.LinkLocalAddress),
			"IPv6 Gateway":  setIP(&out.IPv6Gateway),
			"VLAN Enable":   setBool(&out.VLANEnable),
			"VLAN ID":       setInt(&out.VLANID),
			"VLAN priority": setInt(&out.VLANpriority),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}
	return &out, nil
}