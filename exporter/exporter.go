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

	// server stats
	serverBitrateIn  *prometheus.Desc
	serverBitrateOut *prometheus.Desc
	serverRxTotal    *prometheus.Desc
	serverTxTotal    *prometheus.Desc

	// stream stats
	streamUptimeSeconds *prometheus.Desc
	streamBitrateIn     *prometheus.Desc
	streamBitrateOut    *prometheus.Desc
	streamRxTotal       *prometheus.Desc
	streamTxTotal       *prometheus.Desc
	streamClients       *prometheus.Desc
	streamInfo          *prometheus.Desc
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

		serverBitrateIn: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "server", "bitrate_in"),
			"Current incoming bitrate to the server",
			nil, nil,
		),
		serverBitrateOut: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "server", "bitrate_out"),
			"Current outgoing bitrate from the server",
			nil, nil,
		),
		serverRxTotal: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "server", "bytes_read_total"),
			"Total amount of bytes read by the server",
			nil, nil,
		),
		serverTxTotal: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "server", "bytes_sent_total"),
			"Total amount of bytes sent by the server",
			nil, nil,
		),

		streamUptimeSeconds: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "stream", "uptime_seconds"),
			"Uptime of the stream in seconds",
			[]string{"application", "stream", "publisher"},
			nil,
		),
		streamBitrateIn: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "stream", "bitrate_in"),
			"Current incoming bitrate for the given stream",
			[]string{"application", "stream", "publisher"},
			nil,
		),
		streamBitrateOut: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "stream", "bitrate_out"),
			"Current outgoing bitrate for the given stream",
			[]string{"application", "stream", "publisher"},
			nil,
		),
		streamRxTotal: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "stream", "bytes_read_total"),
			"Total amount of bytes read for the given stream",
			[]string{"application", "stream", "publisher"},
			nil,
		),
		streamTxTotal: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "stream", "bytes_sent_total"),
			"Total amount of bytes sent by the given stream",
			[]string{"application", "stream", "publisher"},
			nil,
		),
		streamClients: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "stream", "current_clients"),
			"Current number of clients connected to the given stream",
			[]string{"application", "stream", "publisher"},
			nil,
		),
		streamInfo: prometheus.NewDesc(
			prometheus.BuildFQName("rtmp", "stream", "info"),
			"Info for a specific stream",
			[]string{"application", "stream", "publisher", "video_resolution", "frame_rate", "video_codec", "audio_codec", "audio_channels", "audio_sample_rate"},
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

	ch <- prometheus.MustNewConstMetric(e.nginxBuildInfo, prometheus.GaugeValue, 1, s.NGINXVersion, s.NGINXRTMPVersion, s.Compiler, s.Built.String())

	ch <- prometheus.MustNewConstMetric(e.serverBitrateIn, prometheus.GaugeValue, float64(s.BitrateIn))
	ch <- prometheus.MustNewConstMetric(e.serverBitrateOut, prometheus.GaugeValue, float64(s.BitrateOut))
	ch <- prometheus.MustNewConstMetric(e.serverRxTotal, prometheus.CounterValue, float64(s.BytesIn))
	ch <- prometheus.MustNewConstMetric(e.serverTxTotal, prometheus.CounterValue, float64(s.BytesOut))

	for _, app := range s.Applications {
		for _, stream := range app.Streams {
			var publisher rtmpstats.Client
			for _, cli := range stream.Clients {
				if cli.Publishing {
					publisher = cli
					break
				}
			}

			ch <- prometheus.MustNewConstMetric(e.streamUptimeSeconds, prometheus.CounterValue, float64(stream.Uptime.Seconds()), app.Name, stream.Name, publisher.ID)
			ch <- prometheus.MustNewConstMetric(e.streamBitrateIn, prometheus.GaugeValue, float64(stream.BitrateIn), app.Name, stream.Name, publisher.ID)
			ch <- prometheus.MustNewConstMetric(e.streamBitrateOut, prometheus.GaugeValue, float64(stream.BitrateOut), app.Name, stream.Name, publisher.ID)
			ch <- prometheus.MustNewConstMetric(e.streamRxTotal, prometheus.CounterValue, float64(stream.BytesIn), app.Name, stream.Name, publisher.ID)
			ch <- prometheus.MustNewConstMetric(e.streamTxTotal, prometheus.CounterValue, float64(stream.BytesOut), app.Name, stream.Name, publisher.ID)
			ch <- prometheus.MustNewConstMetric(e.streamClients, prometheus.GaugeValue, float64(stream.NumClients), app.Name, stream.Name, publisher.ID)

			ch <- prometheus.MustNewConstMetric(e.streamInfo, prometheus.GaugeValue, 1,
				app.Name, stream.Name, publisher.ID,
				fmt.Sprintf("%dx%d", stream.VideoWidth, stream.VideoHeight), fmt.Sprintf("%d", stream.VideoFramerate), stream.VideoCodec,
				stream.AudioCodec, fmt.Sprintf("%d", stream.AudioChannels), fmt.Sprintf("%d", stream.AudioSampleRate),
			)
		}
	}
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
