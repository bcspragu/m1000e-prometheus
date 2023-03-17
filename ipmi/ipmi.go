package ipmi

import (
	"fmt"
	"log"

	"github.com/bougou/go-ipmi"
)

type Client struct {
	user    string
	pass    string
	clients map[string]*ipmi.Client
}

func New(user, pass string) *Client {
	return &Client{
		user:    user,
		pass:    pass,
		clients: make(map[string]*ipmi.Client),
	}
}

func (c *Client) AmbientTemp(host string, port int) (float64, error) {
	// Note: We assume this isn't accessed concurrently for now.

	ic, ok := c.clients[host]
	if !ok {
		log.Printf("initing IPMI to host %q", host)

		tmp, err := ipmi.NewClient(host, port, c.user, c.pass)
		if err != nil {
			return 0, fmt.Errorf("failed to init IPMI client: %w", err)
		}
		tmp.Interface = ipmi.InterfaceLanplus

		if err := tmp.Connect(); err != nil {
			return 0, fmt.Errorf("failed to connect to server at %q over IPMI: %w", host, err)
		}
		ic = tmp
		c.clients[host] = ic
	}

	sdr, err := ic.GetSDRBySensorName("Ambient Temp")
	if err != nil {
		return 0, fmt.Errorf("failed to load sdr ambient temp: %w", err)
	}

	return sdr.Full.ConvertReading(sdr.Full.NominalReadingRaw), nil
}