package exporter

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ClickHouse/clickhouse_exporter/internals/exporters"
	"github.com/ClickHouse/clickhouse_exporter/internals/util"
	"github.com/ClickHouse/clickhouse_exporter/pkg/clickhouse"
	"github.com/ClickHouse/clickhouse_exporter/pkg/configs"
	"github.com/ClickHouse/clickhouse_exporter/pkg/yaml"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	NAMESPACE = "clickhouse" // For Prometheus metrics.
)

// Exporter collects clickhouse stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	basicMetricsExporter exporters.BasicMetricsExporter
	asyncMetricsExporter exporters.AsyncMetricsExporter
	eventMetricsExporter exporters.EventMetricsExporter
	partMetricsExporter  exporters.PartsMetricsExporter
	diskMetricsExporter  exporters.DiskMetricsExporter
	queryMetricsExporter exporters.QueryMetricsExporter

	scrapeFailures prometheus.Counter
	clickConn      clickhouse.ClickhouseConn
}

// NewExporter returns an initialized Exporter.
func NewExporter(configs configs.Configuration) *Exporter {

	uri, err := url.Parse(configs.ClickhouseScrapeURI)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Printf("Scraping %s", configs.ClickhouseScrapeURI)

	queryFilters := yaml.ReadYaml(configs.QueryFiltersPath)

	basicMetricsExporter := exporters.NewBasicMetricsExporter(
		*uri,
		NAMESPACE,
		queryFilters.GetMapObject("basic_exporter"),
	)

	asyncMetricsExporter := exporters.NewAsyncMetricsExporter(
		*uri,
		NAMESPACE,
		queryFilters.GetMapObject("async_exporter"),
	)

	eventMetricsExporter := exporters.NewEventMetricsExporter(
		*uri,
		NAMESPACE,
		queryFilters.GetMapObject("event_exporter"),
	)

	partMetricsExporter := exporters.NewPartsMetricsExporter(
		*uri,
		NAMESPACE,
		queryFilters.GetMapObject("parts_exporter"),
	)

	diskMetricsExporter := exporters.NewDiskMetricsExporter(
		*uri,
		NAMESPACE,
		queryFilters.GetMapObject("disk_exporter"),
	)

	queryMetricsExporter := exporters.NewQueryMetricsExporter(
		*uri,
		NAMESPACE,
		queryFilters.GetMapObject("query_exporter"),
	)

	return &Exporter{
		basicMetricsExporter: basicMetricsExporter,
		asyncMetricsExporter: asyncMetricsExporter,
		eventMetricsExporter: eventMetricsExporter,
		partMetricsExporter:  partMetricsExporter,
		diskMetricsExporter:  diskMetricsExporter,
		queryMetricsExporter: queryMetricsExporter,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: NAMESPACE,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping clickhouse.",
		}),
		clickConn: clickhouse.ClickhouseConn{
			Client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: *configs.Insecure},
				},
				Timeout: 30 * time.Second,
			},
			User:     configs.User,
			Password: configs.Password,
		},
	}
}

// Describe describes all the metrics ever exported by the clickhouse exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	// We cannot know in advance what metrics the exporter will generate
	// from clickhouse. So we use the poor man's describe method: Run a collect
	// and send the descriptors of all the collected metrics.

	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {

	// PARSING RESPONSES
	metrics, err := util.ParseKeyValueResponse(e.basicMetricsExporter.QueryURI, e.clickConn)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.basicMetricsExporter.QueryURI, err)
	}

	asyncMetrics, err := util.ParseKeyValueResponse(e.asyncMetricsExporter.QueryURI, e.clickConn)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.asyncMetricsExporter.QueryURI, err)
	}

	events, err := util.ParseKeyValueResponse(e.eventMetricsExporter.QueryURI, e.clickConn)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.eventMetricsExporter.QueryURI, err)
	}

	parts, err := e.partMetricsExporter.ParsePartsResponse(e.clickConn)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.partMetricsExporter.QueryURI, err)
	}

	disksMetrics, err := e.diskMetricsExporter.ParseDiskResponse(e.clickConn)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.diskMetricsExporter.QueryURI, err)
	}

	query_metrics, err := e.queryMetricsExporter.ParseQueryResponse(e.clickConn)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.diskMetricsExporter.QueryURI, err)
	}

	// COLLECTING METRICS BY PROMETHEUS
	e.basicMetricsExporter.Collect(metrics, ch)
	e.asyncMetricsExporter.Collect(asyncMetrics, ch)
	e.eventMetricsExporter.Collect(events, ch)
	e.partMetricsExporter.Collect(parts, ch)
	e.diskMetricsExporter.Collect(disksMetrics, ch)
	e.queryMetricsExporter.Collect(query_metrics, ch)

	return nil
}

// Collect fetches the stats from configured clickhouse location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	upValue := 1

	if err := e.collect(ch); err != nil {
		log.Error().Msgf("Error scraping clickhouse: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)

		upValue = 0
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(NAMESPACE, "", "up"),
			"Was the last query of ClickHouse successful.",
			nil, nil,
		),
		prometheus.GaugeValue, float64(upValue),
	)

}

// check interface
var _ prometheus.Collector = (*Exporter)(nil)
