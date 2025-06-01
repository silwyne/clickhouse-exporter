package exporter

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ClickHouse/clickhouse_exporter/src/pkg/exporters"
	"github.com/ClickHouse/clickhouse_exporter/src/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	namespace = "clickhouse" // For Prometheus metrics.
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
	clickConn      util.ClickhouseConn
}

// NewExporter returns an initialized Exporter.
func NewExporter(uri url.URL, insecure bool, user, password string) *Exporter {

	basicMetricsExporter := exporters.NewBasicMetricsExporter(
		uri,
		namespace,
	)

	asyncMetricsExporter := exporters.NewAsyncMetricsExporter(
		uri,
		namespace,
	)

	eventMetricsExporter := exporters.NewEventMetricsExporter(
		uri,
		namespace,
	)

	partMetricsExporter := exporters.NewPartsMetricsExporter(
		uri,
		namespace,
	)

	diskMetricsExporter := exporters.NewDiskMetricsExporter(
		uri,
		namespace,
	)

	queryMetricsExporter := exporters.NewQueryMetricsExporter(
		uri,
		namespace,
	)

	return &Exporter{
		basicMetricsExporter: basicMetricsExporter,
		asyncMetricsExporter: asyncMetricsExporter,
		eventMetricsExporter: eventMetricsExporter,
		partMetricsExporter:  partMetricsExporter,
		diskMetricsExporter:  diskMetricsExporter,
		queryMetricsExporter: queryMetricsExporter,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping clickhouse.",
		}),
		clickConn: util.ClickhouseConn{
			Client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
				},
				Timeout: 30 * time.Second,
			},
			User:     user,
			Password: password,
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
		log.Info().Msgf("Error scraping clickhouse: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)

		upValue = 0
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Was the last query of ClickHouse successful.",
			nil, nil,
		),
		prometheus.GaugeValue, float64(upValue),
	)

}

// check interface
var _ prometheus.Collector = (*Exporter)(nil)
