// Package exporter contains a Prometheus Collector that exposes metrics on the
// nginx_rtmp_module.
package exporter

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rfratto/rtmp_exporter/rtmpstats"
	"golang.org/x/net/context"
)

type Config struct {
	StatsURL  string
	StatsFile string
	Timeout   time.Duration
}

func (c *Config) RegisterFlagsWithPrefix(prefix string, fs *flag.FlagSet) {
	fs.StringVar(&c.StatsURL, prefix+"stats-url", "", "URL to get the nginx rtmp stats from")
	fs.StringVar(&c.StatsFile, prefix+"stats-file", "", "File on disk to get the stats file from rather than getting it via URL")
	fs.DurationVar(&c.Timeout, prefix+"stats-timeout", time.Second*5, "timeout to retrieve rtmp stats")
}

// Exporter collects metrics from a nginx rtmp module's stats endpoint.
type Exporter struct {
	cfg      Config
	logger   log.Logger
	mutators []rtmpstats.Mutator

	nginxBuildInfo *prometheus.Desc
}

// New creates a new Exporter.
func New(cfg Config, logger log.Logger, mutators ...rtmpstats.Mutator) *Exporter {
	return &Exporter{
		cfg:      cfg,
		logger:   logger,
		mutators: mutators,

		nginxBuildInfo: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "", "nginx_build_info"),
			"Info about the running nginx server",
			[]string{"nginx_version", "nginx_rtmp_version", "compiler", "built"},
			nil,
		),
	}
}

// Describe describes all the metrics that will be exposed by the rtmp
// exporter. It implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.nginxBuildInfo
}

// Collect fetches the statistics from the configured server, and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	s, err := e.getStats()
	if err != nil {
		level.Error(e.logger).Log("msg", "failed to get stats", "err", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		e.nginxBuildInfo,
		prometheus.GaugeValue,
		1,
		s.NGINXVersion,
		s.NGINXRTMPVersion,
		s.Compiler,
		s.Built.String(),
	)
}

func (e *Exporter) getStats() (*rtmpstats.Stats, error) {
	switch {
	case e.cfg.StatsFile != "":
		return e.getStatsFromFile()
	default:
		return e.getStatsFromURL()
	}
}

func (e *Exporter) getStatsFromFile() (*rtmpstats.Stats, error) {
	f, err := os.Open(e.cfg.StatsFile)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	s, err := rtmpstats.Unmarshal(f, e.mutators...)
	if err != nil {
		return nil, fmt.Errorf("reading stats: %w", err)
	}
	return s, nil
}

func (e *Exporter) getStatsFromURL() (*rtmpstats.Stats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", e.cfg.StatsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	s, err := rtmpstats.Unmarshal(resp.Body, e.mutators...)
	if err != nil {
		return nil, fmt.Errorf("reading stats: %w", err)
	}

	return s, nil
}
