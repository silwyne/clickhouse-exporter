package main

import (
	"net/http"
	"net/url"

	"github.com/ClickHouse/clickhouse_exporter/internals/exporter"
	"github.com/ClickHouse/clickhouse_exporter/pkg/configs"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func main() {

	configurations := configs.LoadConfigs()

	uri, err := url.Parse(configurations.ClickhouseScrapeURI)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Printf("Scraping %s", configurations.ClickhouseScrapeURI)

	registerer := prometheus.DefaultRegisterer
	gatherer := prometheus.DefaultGatherer
	if *configurations.ClickhouseOnly {
		reg := prometheus.NewRegistry()
		registerer = reg
		gatherer = reg
	}

	e := exporter.NewExporter(*uri, *configurations.Insecure, configurations.User, configurations.Password)
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
