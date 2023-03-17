package racadm

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type GetPowerBudgetInfo struct {
	// TODO(brandon): Include any of the other useful output from this command (power supplies, power budget, etc)
	ServerPowerInfo []*ServerPowerInfo
}

type ServerPowerInfo struct {
	SlotNumber int
	ServerName string
	PowerState string
	Allocation string
	Priority   int
	BladeType  string
}

func (c *Client) GetPowerBudgetInfo() (*GetPowerBudgetInfo, error) {
	var resp *GetPowerBudgetInfo
	err := c.runCommand("racadm getpbinfo", func(r io.Reader) error {
		var err error
		if resp, err = parseGetPowerBudgetInfo(r); err != nil {
			return fmt.Errorf("failed to parse output: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// pbBlock indicates what block of output text we're parsing
type pbBlock int

const (
	pbBlockNone pbBlock = iota
	pbBlockPowerBudget
	pbBlockChassisPower
	pbBlockServerPower
)

func parseGetPowerBudgetInfo(r io.Reader) (*GetPowerBudgetInfo, error) {
	var out GetPowerBudgetInfo

	currentBlock := pbBlockNone
	headers := map[string]pbBlock{
		"[Power Budget Status]":                  pbBlockPowerBudget,
		"[Chassis Power Supply Status Table]":    pbBlockChassisPower,
		"[Server Module Power Allocation Table]": pbBlockServerPower,
	}
	err := parseOutput(r, parseConfig{
		splitFn: func(in string) (string, []string, error) {
			txt := strings.TrimSpace(in)
			if txt == "" {
				return "", nil, errSkip
			}

			// This is a description of the columns, a header row.
			if strings.HasPrefix(txt, "<") {
				return "", nil, errSkip
			}

			nextBlock, ok := headers[txt]
			if ok {
				currentBlock = nextBlock
				return "", nil, errSkip
			}

			switch currentBlock {
			case pbBlockPowerBudget, pbBlockChassisPower:
				// Currently unhandled, see TODO on GetPowerBudgetInfo.
				return "", nil, errSkip
			case pbBlockServerPower:
				fs := strings.Fields(txt)
				if len(fs) < 7 {
					return "", nil, fmt.Errorf("got %d, expected 7 fields", len(fs))
				}
				return "servers", fs, nil
			default:
				return "", nil, fmt.Errorf("unknown pb output block %d", currentBlock)
			}
		},
		extractors: map[string]extract{
			"servers": {
				fn: func(vals []string) error {
					sp, err := parseServerPower(vals)
					if err != nil {
						return err
					}
					out.ServerPowerInfo = append(out.ServerPowerInfo, sp)
					return nil
				},
				allowMultiple: true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}
	return &out, nil
}

func parseServerPower(vals []string) (*ServerPowerInfo, error) {
	if len(vals) != 7 {
		return nil, fmt.Errorf("unexpected number of values %d, wanted 7", len(vals))
	}
	num, err := strconv.Atoi(vals[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse server slot number: %w", err)
	}
	prio, err := strconv.Atoi(vals[5])
	if err != nil {
		return nil, fmt.Errorf("failed to parse priority: %w", err)
	}

	return &ServerPowerInfo{
		SlotNumber: num,
		ServerName: vals[1],
		PowerState: vals[2],
		Allocation: vals[3] + " " + vals[4], // e.g. "300 W"
		Priority:   prio,
		BladeType:  vals[6],
	}, nil
}
