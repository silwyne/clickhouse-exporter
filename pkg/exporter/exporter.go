package exporter

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"clickhouse-metric-exporter/pkg/exporters"
	"clickhouse-metric-exporter/pkg/util"

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
	disksMetricURI       string

	scrapeFailures prometheus.Counter
	clickConn      util.ClickhouseConn
}

// NewExporter returns an initialized Exporter.
func NewExporter(uri url.URL, insecure bool, user, password string) *Exporter {

	q := uri.Query()

	basicMetricsExporter := exporters.NewBasicMetricsExporter(
		"select metric, value from system.metrics",
		uri,
		namespace,
	)

	asyncMetricsExporter := exporters.NewAsyncMetricsExporter(
		"select replaceRegexpAll(toString(metric), '-', '_') AS metric, value from system.asynchronous_metrics",
		uri,
		namespace,
	)

	eventMetricsExporter := exporters.NewEventMetricsExporter(
		"select event, value from system.events",
		uri,
		namespace,
	)

	partMetricsExporter := exporters.NewPartsMetricsExporter(
		"select database, table, sum(bytes) as bytes, count() as parts, sum(rows) as rows from system.parts where active = 1 group by database, table",
		uri,
		namespace,
	)

	disksMetricURI := uri
	q.Set("query", `select name, sum(free_space) as free_space_in_bytes, sum(total_space) as total_space_in_bytes from system.disks group by name`)
	disksMetricURI.RawQuery = q.Encode()

	return &Exporter{
		basicMetricsExporter: basicMetricsExporter,
		asyncMetricsExporter: asyncMetricsExporter,
		eventMetricsExporter: eventMetricsExporter,
		partMetricsExporter:  partMetricsExporter,
		disksMetricURI:       disksMetricURI.String(),
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

	disksMetrics, err := e.parseDiskResponse(e.disksMetricURI)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.disksMetricURI, err)
	}

	e.basicMetricsExporter.Collect(metrics, ch)
	e.asyncMetricsExporter.Collect(asyncMetrics, ch)
	e.eventMetricsExporter.Collect(events, ch)
	e.partMetricsExporter.Collect(parts, ch)

	for _, dm := range disksMetrics {
		newFreeSpaceMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "free_space_in_bytes",
			Help:      "Disks free_space_in_bytes capacity",
		}, []string{"disk"}).WithLabelValues(dm.disk)
		newFreeSpaceMetric.Set(dm.freeSpace)
		newFreeSpaceMetric.Collect(ch)

		newTotalSpaceMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "total_space_in_bytes",
			Help:      "Disks total_space_in_bytes capacity",
		}, []string{"disk"}).WithLabelValues(dm.disk)
		newTotalSpaceMetric.Set(dm.totalSpace)
		newTotalSpaceMetric.Collect(ch)
	}

	return nil
}

type diskResult struct {
	disk       string
	freeSpace  float64
	totalSpace float64
}

func (e *Exporter) parseDiskResponse(uri string) ([]diskResult, error) {
	data, err := e.clickConn.ExecuteURI(uri)
	if err != nil {
		return nil, err
	}

	// Parsing results
	lines := strings.Split(string(data), "\n")
	var results = make([]diskResult, 0)

	for i, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) != 3 {
			return nil, fmt.Errorf("parseDiskResponse: unexpected %d line: %s", i, line)
		}
		disk := strings.TrimSpace(parts[0])

		freeSpace, err := util.ParseNumber(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}

		totalSpace, err := util.ParseNumber(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, err
		}

		results = append(results, diskResult{disk, freeSpace, totalSpace})

	}
	return results, nil
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
