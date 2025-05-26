package exporters

import (
	"clickhouse-metric-exporter/pkg/util"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
)

type EventMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewEventMetricsExporter(query string, uri url.URL, namespace string) EventMetricsExporter {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", query)
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
