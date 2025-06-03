package main

import (
	"net/http"

	"github.com/ClickHouse/clickhouse_exporter/internals/exporter"
	"github.com/ClickHouse/clickhouse_exporter/pkg/configs"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func main() {

	configurations := configs.LoadConfigs()

	registerer := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer
	if *configurations.ClickhouseOnly {
		reg := prometheus.NewRegistry()
		registerer = reg
		gatherer = reg
	}

	e := exporter.NewExporter(configurations)
	registerer.MustRegister(e)

	http.Handle(*configurations.MetricsEndpoint, promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Clickhouse Exporter</title></head>
			<body>
			<h1>Clickhouse Exporter</h1>
			<p><a href="` + *configurations.MetricsEndpoint + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Fatal().Err(http.ListenAndServe(*configurations.ListeningAddress, nil)).Send()
}
