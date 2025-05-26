package exporters

import (
	"clickhouse-metric-exporter/pkg/util"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
)

type BasicMetrics struct {
	Namespace string
	QueryURI  string
}

func NewBasicMetric(query string, uri url.URL, namespace string) BasicMetrics {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", "select metric, value from system.metrics")
	metricsURI.RawQuery = url_values.Encode()

	return BasicMetrics{
		QueryURI:  metricsURI.String(),
		Namespace: namespace,
	}
}

func (e *BasicMetrics) Collect(resultLines []util.LineResult, ch chan<- prometheus.Metric) {
	for _, m := range resultLines {
		newMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: e.Namespace,
			Name:      util.GetMetricName(m.Key),
			Help:      "Number of " + m.Key + " currently processed",
		}, []string{}).WithLabelValues()
		newMetric.Set(m.Value)
		newMetric.Collect(ch)
	}
}
