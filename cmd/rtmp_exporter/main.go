package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rfratto/rtmp_exporter/exporter"
	"github.com/weaveworks/common/logging"
)

func main() {
	var (
		cfg        exporter.Config
		listenPort int
		logLevel   logging.Level
	)

	fs := flag.NewFlagSet("rtmp_exporter", flag.ExitOnError)
	fs.IntVar(&listenPort, "listen-port", 8080, "port to listen on to expose /metrics")
	logLevel.RegisterFlags(fs)
	cfg.RegisterFlagsWithPrefix("", fs)

	logger, err := util.NewPrometheusLogger(logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %s", err)
		os.Exit(1)
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		level.Error(logger).Log("msg", "failed to parse flags", "err", err)
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		level.Error(logger).Log("msg", "failed to create listener", "err", err)
		os.Exit(1)
	}

	prometheus.MustRegister(exporter.New(cfg, logger))

	level.Info(logger).Log("msg", "server listening on port", "port", listenPort)
	if err := http.Serve(lis, promhttp.Handler()); err != nil {
		level.Error(logger).Log("msg", "serving failed", "err", err)
		os.Exit(1)
	}
}
