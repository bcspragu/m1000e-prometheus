package racadm

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseGetSensorInfo(t *testing.T) {
	in := strings.NewReader(`
<senType>       <Num>   <sensorName>    <status>        <reading>       <units>         <LC>    <UC>
FanSpeed        1       Fan-1           OK              1000            rpm             1000    14500
FanSpeed        2       Fan-2           OK              2000            rpm             1000    14500
FanSpeed        3       Fan-3           OK              3000            rpm             2000    14500
FanSpeed        4       Fan-4           OK              4000            rpm             1000    14500
FanSpeed        5       Fan-5           OK              5000            rpm             1000    14500
FanSpeed        6       Fan-6           OK              6000            rpm             2000    14500
FanSpeed        7       Fan-7           OK              7000            rpm             2000    9835
FanSpeed        8       Fan-8           OK              8000            rpm             1000    14500
FanSpeed        9       Fan-9           OK              9000            rpm             2000    14500

<senType>       <Num>   <sensorName>    <status>        <reading>       <units>         <LC>    <UC>
Temp            1       Ambient_Temp    OK              20              Celsius         N/A     40

<senType>       <Num>   <sensorName>    <status>        <health>
PWR             1       PS-1            Online          OK
PWR             2       PS-2            Online          OK
PWR             3       PS-3            Online          OK
PWR             4       PS-4            Online          OK
PWR             5       PS-5            Online          OK
PWR             6       PS-6            Online          OK

<senType>       <Num>   <sensorName>    <status>
Cable           1       IO-Cable        OK
Cable           2       FPC-Cable       OK
`)

	got, err := parseGetSensorInfo(in)
	if err != nil {
		t.Fatalf("parseGetSensorInfo: %v", err)
	}

	want := &GetSensorInfo{
		Fans: []*Sensor{
			{Number: 1, SensorName: "Fan-1", Status: "OK", Reading: 1000, Units: "rpm"},
			{Number: 2, SensorName: "Fan-2", Status: "OK", Reading: 2000, Units: "rpm"},
			{Number: 3, SensorName: "Fan-3", Status: "OK", Reading: 3000, Units: "rpm"},
			{Number: 4, SensorName: "Fan-4", Status: "OK", Reading: 4000, Units: "rpm"},
			{Number: 5, SensorName: "Fan-5", Status: "OK", Reading: 5000, Units: "rpm"},
			{Number: 6, SensorName: "Fan-6", Status: "OK", Reading: 6000, Units: "rpm"},
			{Number: 7, SensorName: "Fan-7", Status: "OK", Reading: 7000, Units: "rpm"},
			{Number: 8, SensorName: "Fan-8", Status: "OK", Reading: 8000, Units: "rpm"},
			{Number: 9, SensorName: "Fan-9", Status: "OK", Reading: 9000, Units: "rpm"},
		},
		AmbientTemp: []*Sensor{
			{
				Number:     1,
				SensorName: "Ambient_Temp",
				Status:     "OK",
				Reading:    20,
				Units:      "Celsius",
			},
		},
		PowerSupplies: []*PowerSupplyInfo{
			{Number: 1, SensorName: "PS-1", Status: "Online", Health: "OK"},
			{Number: 2, SensorName: "PS-2", Status: "Online", Health: "OK"},
			{Number: 3, SensorName: "PS-3", Status: "Online", Health: "OK"},
			{Number: 4, SensorName: "PS-4", Status: "Online", Health: "OK"},
			{Number: 5, SensorName: "PS-5", Status: "Online", Health: "OK"},
			{Number: 6, SensorName: "PS-6", Status: "Online", Health: "OK"},
		},
		Cables: []*CableInfo{
			{Number: 1, SensorName: "IO-Cable", Status: "OK"},
			{Number: 2, SensorName: "FPC-Cable", Status: "OK"},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected GetSensorInfo output (-want +got)\n%s", diff)
	}
}

func TestParseGetSysInfo(t *testing.T) {
	in := strings.NewReader(`CMC Information:
CMC Date/Time             = Tue Jan 04 2000 08:51
Primary CMC Location      = CMC-1
Primary CMC Version       = 6.21
Standby CMC Version       = 6.21
Last Firmware Update      = Fri Dec 31 1999 18:31
Hardware Version          = A00

CMC Network Information:
NIC Enabled               = 1
MAC Address               = 00:11:22:33:44:55
Register DNS CMC Name     = 1
DNS CMC Name              = cmc-ABCDEFG
Current DNS Domain        =
VLAN ID                   = 1
VLAN Priority             = 2
VLAN Priority             = 2
VLAN Enabled              = 1

CMC IPv4 Information:
IPv4 Enabled              = 1
Current IP Address        = 192.168.1.2
Current IP Gateway        = 192.168.1.1
Current IP Netmask        = 255.255.255.0
DHCP Enabled              = 1
Current DNS Server 1      = 0.0.0.0
Current DNS Server 2      = 0.0.0.0
DNS Servers from DHCP     = 1

CMC IPv6 Information:
IPv6 Enabled              = 1
Autoconfiguration Enabled = 1
Link Local Address        = ::
Current IPv6 Address 1    = ::
Current IPv6 Gateway      = ::
Current IPv6 DNS Server 1 = ::
Current IPv6 DNS Server 2 = ::
DNS Servers from DHCPv6   = 1

Chassis Information:
System Model              = PowerEdge M1000e
System AssetTag           = 00000
Service Tag               = ABCDEFG
Chassis Name              = CMC-ABCDEFG
Chassis Location          = [UNDEFINED]
Chassis Midplane Version  = 1.0
Power Status              = ON
System ID                 = 1234
`)

	got, err := parseGetSysInfo(in)
	if err != nil {
		t.Fatalf("parseGetSysInfo: %v", err)
	}

	want := &GetSysInfo{
		CMCDateTime:              time.Date(2000, time.January, 4, 8, 51, 0, 0, pst),
		PrimaryCMCLocation:       "CMC-1",
		PrimaryCMCVersion:        "6.21",
		StandbyCMCVersion:        "6.21",
		LastFirmwareUpdate:       time.Date(1999, time.December, 31, 18, 31, 0, 0, pst),
		HardwareVersion:          "A00",
		NICEnabled:               true,
		MACAddress:               parseMAC(t, "00:11:22:33:44:55"),
		RegisterDNSCMCName:       true,
		DNSCMCName:               "cmc-ABCDEFG",
		CurrentDNSDomain:         "",
		VLANID:                   1,
		VLANPriority:             2,
		VLANEnabled:              true,
		IPv4Enabled:              true,
		CurrentIPAddress:         parseIP(t, "192.168.1.2"),
		CurrentIPGateway:         parseIP(t, "192.168.1.1"),
		CurrentIPNetmask:         parseIPMask(t, "255.255.255.0"),
		DHCPEnabled:              true,
		CurrentDNSServer1:        parseIP(t, "0.0.0.0"),
		CurrentDNSServer2:        parseIP(t, "0.0.0.0"),
		DNSServersfromDHCP:       true,
		IPv6Enabled:              true,
		AutoconfigurationEnabled: true,
		LinkLocalAddress:         parseIP(t, "::"),
		CurrentIPv6Address1:      parseIP(t, "::"),
		CurrentIPv6Gateway:       parseIP(t, "::"),
		CurrentIPv6DNSServer1:    parseIP(t, "::"),
		CurrentIPv6DNSServer2:    parseIP(t, "::"),
		DNSServersfromDHCPv6:     true,
		SystemModel:              "PowerEdge M1000e",
		SystemAssetTag:           "00000",
		ServiceTag:               "ABCDEFG",
		ChassisName:              "CMC-ABCDEFG",
		ChassisLocation:          "[UNDEFINED]",
		ChassisMidplaneVersion:   "1.0",
		PowerStatus:              "ON",
		SystemID:                 "1234",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected GetSysInfo output (-want +got)\n%s", diff)
	}
}

func TestParseGetPowerBudgetInfo(t *testing.T) {
	in := strings.NewReader(`

[Power Budget Status]
System Input Power                              = 2345 W
Peak System Power                               = 3456 W
Peak System Power Timestamp                     = 23:05:06 01/04/2000
Minimum System Power                            = 1000 W
Minimum System Power Timestamp                  = 18:02:55 12/31/1999
Overall Power Health                            = OK
Redundancy                                      = Yes
System Input Power Cap                          = 16786 W
Redundancy Policy                               = None
Dynamic PSU Engagement Enabled                  = Yes
System Input Max Power Capacity                 = 15678 W
Input Redundancy Reserve                        = 0 W
Input Power Allocated to Servers                = 100 W
Input Power Allocated to Chassis Infrastructure = 678 W
Total Input Power Available for Allocation      = 12456 W
Standby Input Power Capacity                    = 0 W
Server Based Power Management Mode              = Yes
Max Power Conservation Mode                     = Yes
Server Performance Over Power Redundancy        = Yes
Power Available for Server Power-on             = 15432 W
Extended Power Performance(EPP) Status          = Disabled
Available Power in EPP Pool                     = 0 W (0 BTU/h)
Used Power in EPP Pool                          = 0 W (0 BTU/h)
EPP Percent - Available                         = 0.0

[Chassis Power Supply Status Table]
<Name>          <Model>         <Power State>          <Input Current> <Input Volts>   <Output Rated Power>
PS1             111111          Online                 1.3 A                  239.1 V                2360 W
PS2             222222          Online                 0.2 A                  238.2 V                2360 W
PS3             333333          Online                 0.3 A                  240.3 V                2360 W
PS4             444444          Online                 1.3 A                  238.4 V                2360 W
PS5             555555          Online                 1.5 A                  241.5 V                2360 W
PS6             666666          Online                 1.3 A                  239.6 V                2360 W

[Server Module Power Allocation Table]
<Slot#> <Server Name>  <Power State>   <Allocation>    <Priority>  <Blade Type>
1       SLOT-01         OFF             0 W             1           PowerEdgeM610
2       SLOT-02         OFF             0 W             1           PowerEdgeM610
3       SLOT-03         OFF             0 W             1           PowerEdgeM610
4       SLOT-04         OFF             0 W             1           PowerEdgeM610
5       SLOT-05         OFF             0 W             1           PowerEdgeM610
6       SLOT-06         OFF             0 W             1           PowerEdgeM610
7       SLOT-07         OFF             0 W             1           PowerEdgeM610
8       SLOT-08         OFF             0 W             1           PowerEdgeM610
9       SLOT-09         OFF             0 W             1           PowerEdgeM610
10      SLOT-10         OFF             0 W             1           PowerEdgeM610
11      SLOT-11         OFF             0 W             1           PowerEdgeM610
12      SLOT-12         OFF             0 W             1           PowerEdgeM610
13      SLOT-13         OFF             0 W             1           PowerEdgeM610
14      SLOT-14         ON              323 W           1           PowerEdgeM610
15      SLOT-15         ON              323 W           1           PowerEdgeM610
16      SLOT-16         ON              316 W           1           PowerEdgeM610
`)

	got, err := parseGetPowerBudgetInfo(in)
	if err != nil {
		t.Fatalf("parseGetPowerBudgetInfo: %v", err)
	}

	want := &GetPowerBudgetInfo{
		ServerPowerInfo: []*ServerPowerInfo{
			{SlotNumber: 1, ServerName: "SLOT-01", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 2, ServerName: "SLOT-02", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 3, ServerName: "SLOT-03", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 4, ServerName: "SLOT-04", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 5, ServerName: "SLOT-05", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 6, ServerName: "SLOT-06", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 7, ServerName: "SLOT-07", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 8, ServerName: "SLOT-08", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 9, ServerName: "SLOT-09", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 10, ServerName: "SLOT-10", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 11, ServerName: "SLOT-11", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 12, ServerName: "SLOT-12", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 13, ServerName: "SLOT-13", PowerState: "OFF", Allocation: "0 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 14, ServerName: "SLOT-14", PowerState: "ON", Allocation: "323 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 15, ServerName: "SLOT-15", PowerState: "ON", Allocation: "323 W", Priority: 1, BladeType: "PowerEdgeM610"},
			{SlotNumber: 16, ServerName: "SLOT-16", PowerState: "ON", Allocation: "316 W", Priority: 1, BladeType: "PowerEdgeM610"},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected GetPowerBudgetInifo output (-want +got)\n%s", diff)
	}
}

func TestParseGetNICInfo(t *testing.T) {
	in := strings.NewReader(`LOM Model Name            = Embedded LOM
LOM Fabric Type           = Gigabit Ethernet
IPv4 Enabled              = 1
DHCP Enabled              = 1
IP Address                = 192.168.2.16
Subnet Mask               = 255.255.255.0
Gateway                   = 192.168.2.1
IPv6 Enabled              = 1
Autoconfiguration Enabled = 1
Link local Address        =
IPv6 Gateway              = ::
VLAN Enable               = 1
VLAN ID                   = 2
VLAN priority             = 3
`)

	got, err := parseGetNICConfig(in)
	if err != nil {
		t.Fatalf("parseGetNICConfg: %v", err)
	}

	want := &GetNICConfig{
		LOMModelName:             "Embedded LOM",
		LOMFabricType:            "Gigabit Ethernet",
		IPv4Enabled:              true,
		DHCPEnabled:              true,
		IPAddress:                parseIP(t, "192.168.2.16"),
		SubnetMask:               parseIPMask(t, "255.255.255.0"),
		Gateway:                  parseIP(t, "192.168.2.1"),
		IPv6Enabled:              true,
		AutoconfigurationEnabled: true,
		// LinkLocalAddress:         parseIP(t, "::"),
		IPv6Gateway:  parseIP(t, "::"),
		VLANEnable:   true,
		VLANID:       2,
		VLANpriority: 3,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected GetNICConfig output (-want +got)\n%s", diff)
	}
}

func parseMAC(t *testing.T, in string) net.HardwareAddr {
	t.Helper()
	hw, err := net.ParseMAC(in)
	if err != nil {
		t.Fatalf("net.ParseMAC: %v", err)
	}
	return hw
}

func parseIP(t *testing.T, in string) net.IP {
	t.Helper()
	ip := net.ParseIP(in)
	if ip == nil {
		t.Fatalf("invalid IP %q", in)
	}
	return ip
}

func parseIPMask(t *testing.T, in string) net.IPMask {
	t.Helper()
	ip := parseIP(t, in)
	return net.IPMask(ip)
}
