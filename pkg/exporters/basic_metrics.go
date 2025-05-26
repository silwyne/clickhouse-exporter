package exporters

import (
	"clickhouse-metric-exporter/pkg/util"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
)

type BasicMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewBasicMetricsExporter(query string, uri url.URL, namespace string) BasicMetricsExporter {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", query)
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
