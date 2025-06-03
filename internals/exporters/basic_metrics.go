package exporters

import (
	"net/url"
	"strings"

	"github.com/ClickHouse/clickhouse_exporter/internals/util"
	"github.com/ClickHouse/clickhouse_exporter/pkg/queryparser"
	"github.com/ClickHouse/clickhouse_exporter/pkg/yaml"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	BASIC_METRIC_EXPORTER_QUERY = "select metric, value from system.metrics {FILTER_CLAUSE}"
)

type BasicMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewBasicMetricsExporter(uri url.URL, namespace string, yamlconfig yaml.YamlConfig) BasicMetricsExporter {

	filter_calause := queryparser.ParseYamlConfigToQueryFilter(yamlconfig)
	query := strings.Replace(BASIC_METRIC_EXPORTER_QUERY, "{FILTER_CLAUSE}", filter_calause, 1)
	log.Printf("metrics exporter query: %v", query)

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
