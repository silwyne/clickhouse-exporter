package exporters

import (
	"clickhouse-metric-exporter/pkg/util"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
)

type AsyncMetrics struct {
	Namespace string
	QueryURI  string
}

func NewAsyncMetrics(query string, uri url.URL, namespace string) AsyncMetrics {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", "select metric, value from system.metrics")
	metricsURI.RawQuery = url_values.Encode()

	return AsyncMetrics{
		QueryURI:  metricsURI.String(),
		Namespace: namespace,
	}
}

func (e *AsyncMetrics) Collect(resultLines []util.LineResult, ch chan<- prometheus.Metric) {
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
