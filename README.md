# Dell M1000e Prometheus

I picked up a [Dell M1000e](https://en.wikipedia.org/wiki/Dell_M1000e) on Craigslist because good decision making isn't my strong suit, and I wanted to monitor it with Prometheus to make sure the basement isn't on fire, so here we are.

This project connects to the CMC over SSH and loads the following data:

* `racadm getsysinfo` - Used as a 'ping' to make sure our connection is good
  * Could also be used to share other metrics about the chassis, see `GetSysInfo`.
* `racadm getsensorinfo` - Used to get ambient chassis temp and per-server fan speeds
* `racadm getpbinfo` - Used to find out which servers are currently on.
  * There are probably other ways to do this, but this works fine.
* `racadm getnicconfig -m server -X` - Used to get the IP of an individual server

Infuriatingly, while individual server temps are available in the CMC Web UI, I could find no way to query them with RACADM ([relevant thread](https://www.dell.com/community/Systems-Management-General/Getting-ambient-temperature-from-iDRAC/m-p/3577536)). And instead of scraping/emulating the web UI ([which some projects do](https://github.com/11harveyj/idrac6-api)), I decided to get the info over the IPMI interface. You can enable this by SSHing into iDRAC on an individual blade and running:

```bash
racadm config -g cfgIpmiLan -o cfgIpmiLanEnable 1
```

When all is said and done, the exported metrics look something like:

```
# HELP m1000e_ambient_temp_celsius Current ambient temperature of the chassis.
# TYPE m1000e_ambient_temp_celsius gauge
m1000e_ambient_temp_celsius{name="Ambient_Temp",number="1",status="OK"} 18
# HELP m1000e_fan_rpm The speed the fans are spinning.
# TYPE m1000e_fan_rpm gauge
m1000e_fan_rpm{name="Fan-X",number="X",status="OK"} 6997
[ ... more fans ... ]
# HELP m1000e_server_temp_celsius Current ambient temperature of a blade server
# TYPE m1000e_server_temp_celsius gauge
m1000e_server_temp_celsius{blade_type="PowerEdgeM610",name="SLOT-X",power_state="ON",slot_number="X"} 20
[ ... more blade temps ... ]
# HELP promhttp_metric_handler_errors_total Total number of internal errors encountered by the promhttp metric handler.
# TYPE promhttp_metric_handler_errors_total counter
promhttp_metric_handler_errors_total{cause="encoding"} 0
promhttp_metric_handler_errors_total{cause="gathering"} 0
```

## Running

To run the server:

```bash
go run main.go <path to credentials>
```

Where the credentials are formatted as:

```json
{
  "user": "<user>",
  "password": "<pass>",
  "addr": "<ip>:<port, usually 22>",
  "ipmi": {
    "user": "<server user>",
    "password": "<server password, usually 'calvin'>" 
  }
}
```

## Docker

A Docker image is also provided, you can build it with:

```bash
docker build -t bcspragu/m1000e-prometheus .
```

I currently don't have pre-built images anywhere, but if anyone wants them, I'll do that.

## Known Limitations

* Requires IPMI enabled on individual servers
* Only supports user/pass credentials for SSH
  * I'm not even sure if key-based SSH creds are supported, but there is a `racadm sshpkauth` command that seems promising?
