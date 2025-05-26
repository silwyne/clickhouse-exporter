package exporters

import (
	"net/url"

	"github.com/ClickHouse/clickhouse_exporter/src/pkg/util"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	BASIC_METRIC_EXPORTER_QUERY = "select metric, value from system.metrics"
)

type BasicMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewBasicMetricsExporter(uri url.URL, namespace string) BasicMetricsExporter {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", BASIC_METRIC_EXPORTER_QUERY)
	metricsURI.RawQuery = url_values.Encode()

	return BasicMetricsExporter{
		QueryURI:  metricsURI.String(),
		Namespace: namespace,
	}
}

func (e *BasicMetricsExporter) Collect(resultLines []util.LineResult, ch chan<- prometheus.Metric) {
	for _, am := range resultLines {
		newMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: e.Namespace,
			Name:      util.GetMetricName(am.Key),
			Help:      "Number of " + am.Key + " async processed",
		}, []string{}).WithLabelValues()
		newMetric.Set(am.Value)
		newMetric.Collect(ch)
	}
}
