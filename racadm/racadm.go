package racadm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

var pst *time.Location

func init() {
	tmp, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(fmt.Errorf("failed to load America/Los_Angeles timezone: %v", err))
	}
	pst = tmp
}

type Client struct {
	user string
	pass string
	addr string

	done chan struct{}

	mu     sync.RWMutex
	client *ssh.Client
}

func Dial(user, pass, addr string) (*Client, error) {
	c := &Client{user: user, pass: pass, addr: addr, done: make(chan struct{})}
	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	go c.refreshSession()
	return c, nil
}

func (c *Client) refreshSession() {
	t := time.NewTicker(25 * time.Minute)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			c.mu.Lock()
			if err := c.client.Close(); err != nil {
				log.Printf("failed to close old client while refreshing: %v", err)
			}
			if err := c.connect(); err != nil {
				log.Printf("failed to open new connection: %v", err)
			}
			c.mu.Unlock()
		case <-c.done:
			return
		}
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) runCommand(cmd string, fn func(r io.Reader) error) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sess, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer sess.Close()

	stdout, err := sess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	if err := sess.Start(cmd); err != nil {
		return fmt.Errorf("failed to run racadm: %w", err)
	}

	if err := fn(stdout); err != nil {
		return fmt.Errorf("error in parse fn: %w", err)
	}

	if err := sess.Wait(); err != nil {
		return fmt.Errorf("error while waiting for racadm command to complete: %w", err)
	}

	return nil
}

func (c *Client) connect() error {
	client, err := ssh.Dial("tcp", c.addr, &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.pass),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// TODO: Consider allowing enforcing the key that comes back here.
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to SSH: %w", err)
	}
	c.client = client
	return nil
}

type parseConfig struct {
	splitFn func(string) (string, []string, error)

	extractors map[string]extract
}

type extractFn func(vals []string) error

type extract struct {
	fn            extractFn
	allowMultiple bool
}

func singleValueExtract(fn func(in string) error) extract {
	return extract{
		fn: func(vals []string) error {
			if len(vals) != 1 {
				return fmt.Errorf("got %d values, expected exactly one", len(vals))
			}
			return fn(vals[0])
		},
	}
}

func setString(v *string) extract {
	return singleValueExtract(func(in string) error {
		*v = in
		return nil
	})
}

func setBool(v *bool) extract {
	return singleValueExtract(func(in string) error {
		switch in {
		case "0":
			*v = false
		case "1":
			*v = true
		default:
			return fmt.Errorf("unexpected bool value %q", in)
		}
		return nil
	})
}

func setTime(v *time.Time) extract {
	return singleValueExtract(func(in string) error {
		t, err := time.ParseInLocation("Mon Jan 02 2006 15:04", in, pst)
		if err != nil {
			return fmt.Errorf("failed to parse time: %w", err)
		}
		*v = t
		return nil
	})
}

func setInt(v *int) extract {
	return singleValueExtract(func(in string) error {
		n, err := strconv.Atoi(in)
		if err != nil {
			return fmt.Errorf("failed to parse int: %w", err)
		}
		*v = n
		return nil
	})
}

func setMAC(v *net.HardwareAddr) extract {
	return singleValueExtract(func(in string) error {
		hw, err := net.ParseMAC(in)
		if err != nil {
			return fmt.Errorf("failed to parse mac: %w", err)
		}
		*v = hw
		return nil
	})
}

func setIP(v *net.IP) extract {
	return singleValueExtract(func(in string) error {
		ip := net.ParseIP(in)
		if ip == nil {
			return fmt.Errorf("invalid IP %q", ip)
		}
		*v = ip
		return nil
	})
}

func setIPMask(v *net.IPMask) extract {
	return singleValueExtract(func(in string) error {
		ip := net.ParseIP(in)
		if ip == nil {
			return fmt.Errorf("invalid IP %q", ip)
		}
		*v = net.IPMask(ip)
		return nil
	})
}

var errSkip = errors.New("skip")

func parseOutput(r io.Reader, cfg parseConfig) error {
	sc := bufio.NewScanner(r)

	seen := make(map[string]bool)
	for sc.Scan() {
		key, vals, err := cfg.splitFn(sc.Text())
		if errors.Is(err, errSkip) {
			continue
		} else if err != nil {
			return fmt.Errorf("failed to split row: %w", err)
		}
		ex, ok := cfg.extractors[key]
		if !ok {
			continue
		}
		if seen[key] && !ex.allowMultiple {
			return fmt.Errorf("key %q occurred at least twice", key)
		}
		if err := ex.fn(vals); err != nil {
			return fmt.Errorf("failed to extract value(s) from %v for key %q: %w", vals, key, err)
		}
	}

	return nil
}