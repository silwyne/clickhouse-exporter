package exporters

import (
	"net/url"

	"github.com/ClickHouse/clickhouse_exporter/internals/util"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ASYNC_METRIC_EXPORTER_QUERY = "select replaceRegexpAll(toString(metric), '-', '_') AS metric, value from system.asynchronous_metrics"
)

type AsyncMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewAsyncMetricsExporter(uri url.URL, namespace string) AsyncMetricsExporter {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", ASYNC_METRIC_EXPORTER_QUERY)
	metricsURI.RawQuery = url_values.Encode()

	return AsyncMetricsExporter{
		QueryURI:  metricsURI.String(),
		Namespace: namespace,
	}
}

func (e *AsyncMetricsExporter) Collect(resultLines []util.LineResult, ch chan<- prometheus.Metric) {
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
