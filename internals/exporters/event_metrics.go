package exporters

import (
	"net/url"

	"github.com/ClickHouse/clickhouse_exporter/internals/util"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	EVENT_METRIC_EXPORTER_QUERY = "select event, value from system.events"
)

type EventMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewEventMetricsExporter(uri url.URL, namespace string) EventMetricsExporter {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", EVENT_METRIC_EXPORTER_QUERY)
	metricsURI.RawQuery = url_values.Encode()

	return EventMetricsExporter{
		QueryURI:  metricsURI.String(),
		Namespace: namespace,
	}
}

func (e *EventMetricsExporter) Collect(resultLines []util.LineResult, ch chan<- prometheus.Metric) {
	for _, ev := range resultLines {
		newMetric, _ := prometheus.NewConstMetric(
			prometheus.NewDesc(
				e.Namespace+"_"+util.GetMetricName(ev.Key)+"_total",
				"Number of "+ev.Key+" total processed", []string{}, nil),
			prometheus.CounterValue, float64(ev.Value))
		ch <- newMetric
	}
}
