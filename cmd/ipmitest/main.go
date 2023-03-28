package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/bcspragu/m1000e-prom/ipmi"
)

func main() {
	if err := run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type creds struct {
	User     string
	Password string
	Addr     string
	IPMI     *ipmiCreds
}

type ipmiCreds struct {
	User     string
	Password string
}

func run(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: ./ipmitest <path to creds file> <host ip>")
	}
	dat, err := ioutil.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("failed to read creds file: %w", err)
	}
	var crds creds
	if err := json.Unmarshal(dat, &crds); err != nil {
		return fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	c := ipmi.New(crds.IPMI.User, crds.IPMI.Password)
	defer func() {
		if err := c.Close(); err != nil {
			log.Printf("error closing session: %v", err)
		}
	}()

	temp, err := c.AmbientTemp(args[2], 623)
	fmt.Println(temp, err)

	return nil
}
