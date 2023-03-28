package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/bcspragu/m1000e-prom/ipmi"
	"github.com/bcspragu/m1000e-prom/racadm"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

type metrics struct {
	ambientTemp *prometheus.GaugeVec
	fanRPM      *prometheus.GaugeVec
	serverTemp  *prometheus.GaugeVec
}

func newMetrics(reg prometheus.Registerer) (*metrics, error) {
	m := &metrics{
		ambientTemp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "m1000e_ambient_temp_celsius",
				Help: "Current ambient temperature of the chassis.",
			},
			[]string{"number", "name", "status"},
		),
		fanRPM: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "m1000e_fan_rpm",
				Help: "The speed the fans are spinning.",
			},
			[]string{"number", "name", "status"},
		),
		serverTemp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "m1000e_server_temp_celsius",
				Help: "Current ambient temperature of a blade server",
			},
			[]string{"slot_number", "name", "power_state", "blade_type"},
		),
	}
	cols := []prometheus.Collector{
		m.ambientTemp,
		m.fanRPM,
		m.serverTemp,
	}
	for _, col := range cols {
		if err := reg.Register(col); err != nil {
			return nil, fmt.Errorf("failed to register metric: %w", err)
		}
	}
	return m, nil
}

type metricClient struct {
	client        *racadm.Client
	metrics       *metrics
	serverIPCache map[string]net.IP
	ipmi          *ipmi.Client
}

func (mc *metricClient) updateMetrics() {
	mc.updateSensorMetrics()
	mc.updateIPMIMetrics()
}

func (mc *metricClient) updateSensorMetrics() {
	sInfo, err := mc.client.GetSensorInfo()
	if err != nil {
		mc.metrics.ambientTemp.Reset()
		mc.metrics.fanRPM.Reset()
		log.Printf("failed to load sensor info: %v", err)
		return
	}
	for _, s := range sInfo.AmbientTemp {
		labels := prometheus.Labels{
			"number": strconv.Itoa(s.Number),
			"name":   s.SensorName,
			"status": s.Status,
		}
		if s.Units != "Celsius" {
			mc.metrics.ambientTemp.Delete(labels)
			log.Printf("unexpected ambient temp units %q, skipping", s.Units)
			continue
		}
		mc.metrics.ambientTemp.With(labels).Set(float64(s.Reading))
	}

	for _, s := range sInfo.Fans {
		labels := prometheus.Labels{
			"number": strconv.Itoa(s.Number),
			"name":   s.SensorName,
			"status": s.Status,
		}
		if s.Units != "rpm" {
			mc.metrics.fanRPM.Delete(labels)
			log.Printf("unexpected fan speed units %q, skipping", s.Units)
			continue
		}
		mc.metrics.fanRPM.With(labels).Set(float64(s.Reading))
	}
}

func (mc *metricClient) updateIPMIMetrics() {
	pbInfo, err := mc.client.GetPowerBudgetInfo()
	if err != nil {
		mc.metrics.serverTemp.Reset()
		log.Printf("failed to load power budget info: %v", err)
		return
	}

	for _, s := range pbInfo.ServerPowerInfo {
		if s.PowerState != "ON" {
			continue
		}
		labels := prometheus.Labels{
			"slot_number": strconv.Itoa(s.SlotNumber),
			"name":        s.ServerName,
			"power_state": s.PowerState,
			"blade_type":  s.BladeType,
		}

		ip, ok := mc.serverIPCache[s.ServerName]
		if !ok {
			log.Printf("looking up iDRAC IP for server %q", s.ServerName)
			nicConfig, err := mc.client.GetNICConfig(s.SlotNumber)
			if err != nil {
				log.Printf("failed to get NIC config for slot %d: %v", s.SlotNumber, err)
				mc.metrics.serverTemp.Delete(labels)
				continue
			}
			ip = nicConfig.IPAddress
			mc.serverIPCache[s.ServerName] = ip
		}

		temp, err := mc.ipmi.AmbientTemp(ip.String(), 623 /* default IPMI port */)
		if err != nil {
			log.Printf("failed to get temp over IMP for slot %d: %v", s.SlotNumber, err)
			mc.metrics.serverTemp.Delete(labels)
			continue
		}

		mc.metrics.serverTemp.With(labels).Set(temp)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ./chassis-prom <path to creds file>")
	}
	dat, err := ioutil.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("failed to read creds file: %w", err)
	}
	var crds creds
	if err := json.Unmarshal(dat, &crds); err != nil {
		return fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	c, err := racadm.Dial(crds.User, crds.Password, crds.Addr)
	if err != nil {
		return fmt.Errorf("failed to init racadm client: %w", err)
	}
	defer c.Close()

	// Make sure our connection works.
	log.Println("Testing connection...")
	info, err := c.GetSysInfo()
	if err != nil {
		return fmt.Errorf("failed to load sys info: %v", err)
	}
	log.Printf("Connected to chassis %q", info.ChassisName)

	reg := prometheus.NewRegistry()
	m, err := newMetrics(reg)
	if err != nil {
		return fmt.Errorf("failed to init metrics: %w", err)
	}

	ipmiClient := ipmi.New(crds.IPMI.User, crds.IPMI.Password)

	mc := metricClient{
		client:        c,
		metrics:       m,
		serverIPCache: make(map[string]net.IP),
		ipmi:          ipmiClient,
	}

	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		mc.updateMetrics()
		for {
			select {
			case <-t.C:
				mc.updateMetrics()
			case <-done:
				return
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	server := &http.Server{Addr: ":8080", Handler: mux}

	// We buffer the channel because server.Shutdown will cause an error to be
	// thrown, but we aren't listening at that point, so it'll block.
	errChan := make(chan error, 1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil {
			errChan <- fmt.Errorf("failed to run server: %w", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	select {
	case err := <-errChan:
		log.Printf("an error occurred, shutting down: %v", err)
	case <-sig:
		log.Println("signal received, shutting down")
	}

	close(done)
	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("error during server shutdown: %v", err)
	}

	log.Println("waiting for HTTP server + poller to shut down")
	wg.Wait()
	log.Println("shutting down")

	return nil
}
