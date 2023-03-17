package racadm

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type GetSensorInfo struct {
	Fans          []*Sensor
	AmbientTemp   []*Sensor
	PowerSupplies []*PowerSupplyInfo
	Cables        []*CableInfo
}

type Sensor struct {
	Number     int
	SensorName string
	Status     string
	Reading    int
	Units      string
}

type PowerSupplyInfo struct {
	Number     int
	SensorName string
	Status     string
	Health     string
}

type CableInfo struct {
	Number     int
	SensorName string
	Status     string
}

func (c *Client) GetSensorInfo() (*GetSensorInfo, error) {
	var resp *GetSensorInfo
	err := c.runCommand("racadm getsensorinfo", func(r io.Reader) error {
		var err error
		if resp, err = parseGetSensorInfo(r); err != nil {
			return fmt.Errorf("failed to parse output: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func parseSensor(vals []string) (*Sensor, error) {
	if len(vals) != 7 {
		return nil, fmt.Errorf("unexpected number of values %d, wanted 7", len(vals))
	}
	num, err := strconv.Atoi(vals[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse sensor number: %w", err)
	}
	reading, err := strconv.Atoi(vals[3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse sensor reading: %w", err)
	}

	return &Sensor{
		Number:     num,
		SensorName: vals[1],
		Status:     vals[2],
		Reading:    reading,
		Units:      vals[4],
	}, nil
}

func parsePower(vals []string) (*PowerSupplyInfo, error) {
	if len(vals) != 4 {
		return nil, fmt.Errorf("unexpected number of values %d, wanted 4", len(vals))
	}
	num, err := strconv.Atoi(vals[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse sensor number: %w", err)
	}

	return &PowerSupplyInfo{
		Number:     num,
		SensorName: vals[1],
		Status:     vals[2],
		Health:     vals[3],
	}, nil
}

func parseCable(vals []string) (*CableInfo, error) {
	if len(vals) != 3 {
		return nil, fmt.Errorf("unexpected number of values %d, wanted 3", len(vals))
	}
	num, err := strconv.Atoi(vals[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse sensor number: %w", err)
	}

	return &CableInfo{
		Number:     num,
		SensorName: vals[1],
		Status:     vals[2],
	}, nil
}

func parseGetSensorInfo(r io.Reader) (*GetSensorInfo, error) {
	var out GetSensorInfo
	err := parseOutput(r, parseConfig{
		splitFn: func(in string) (string, []string, error) {
			txt := strings.TrimSpace(in)
			if txt == "" {
				return "", nil, errSkip
			}
			if strings.HasPrefix(txt, "<") {
				return "", nil, errSkip
			}
			fs := strings.Fields(txt)
			if len(fs) < 2 {
				return "", nil, fmt.Errorf("expected at least two fields, got %d", len(fs))
			}
			return fs[0], fs[1:], nil
		},
		extractors: map[string]extract{
			"FanSpeed": {
				fn: func(vals []string) error {
					s, err := parseSensor(vals)
					if err != nil {
						return err
					}
					out.Fans = append(out.Fans, s)
					return nil
				},
				allowMultiple: true,
			},
			"Temp": {
				fn: func(vals []string) error {
					s, err := parseSensor(vals)
					if err != nil {
						return err
					}
					out.AmbientTemp = append(out.AmbientTemp, s)
					return nil
				},
				allowMultiple: true,
			},
			"PWR": {
				fn: func(vals []string) error {
					p, err := parsePower(vals)
					if err != nil {
						return err
					}
					out.PowerSupplies = append(out.PowerSupplies, p)
					return nil
				},
				allowMultiple: true,
			},
			"Cable": {
				fn: func(vals []string) error {
					c, err := parseCable(vals)
					if err != nil {
						return err
					}
					out.Cables = append(out.Cables, c)
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
