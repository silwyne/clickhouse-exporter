package exporters

import (
	"clickhouse-metric-exporter/pkg/exporter/util"
	"fmt"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
)

type BasicMetrics struct {
	clickConn util.ClickhouseConn
	namespace string
	queryURI  string
}

func NewBasicMetric(query string, uri url.URL, namespace string) BasicMetrics {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", "select metric, value from system.metrics")
	metricsURI.RawQuery = url_values.Encode()

	return BasicMetrics{
		queryURI:  metricsURI.String(),
		namespace: namespace,
	}
}

func (e *BasicMetrics) Collect(ch chan<- prometheus.Metric) error {
	metrics, err := e.clickConn.ParseKeyValueResponse(e.queryURI)
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", e.queryURI, err)
	}

	for _, m := range metrics {
		newMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: e.namespace,
			Name:      util.GetMetricName(m.key),
			Help:      "Number of " + m.key + " currently processed",
		}, []string{}).WithLabelValues()
		newMetric.Set(m.value)
		newMetric.Collect(ch)
	}

	return nil
}
