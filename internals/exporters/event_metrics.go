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
	EVENT_METRIC_EXPORTER_QUERY = `select event, value from system.events {FILTER_CLAUSE}`
)

type EventMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewEventMetricsExporter(uri url.URL, namespace string, yamlconfig yaml.YamlConfig) EventMetricsExporter {

	filter_calause := queryparser.ParseYamlConfigToQueryFilter(yamlconfig)
	query := strings.Replace(EVENT_METRIC_EXPORTER_QUERY, "{FILTER_CLAUSE}", filter_calause, 1)
	log.Printf("events exporter query: %v", query)

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
