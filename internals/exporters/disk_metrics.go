package exporters

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ClickHouse/clickhouse_exporter/internals/util"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	DISK_METRIC_EXPORTER_QUERY = "select name, sum(free_space) as free_space_in_bytes, sum(total_space) as total_space_in_bytes from system.disks group by name"
)

type DiskMetricsExporter struct {
	Namespace string
	QueryURI  string
}

func NewDiskMetricsExporter(uri url.URL, namespace string) DiskMetricsExporter {

	url_values := uri.Query()

	metricsURI := uri
	url_values.Set("query", DISK_METRIC_EXPORTER_QUERY)
	metricsURI.RawQuery = url_values.Encode()

	return DiskMetricsExporter{
		QueryURI:  metricsURI.String(),
		Namespace: namespace,
	}
}

type diskResult struct {
	disk       string
	freeSpace  float64
	totalSpace float64
}

func (e *DiskMetricsExporter) ParseDiskResponse(clickConn util.ClickhouseConn) ([]diskResult, error) {
	data, err := clickConn.ExecuteURI(e.QueryURI)
	if err != nil {
		return nil, err
	}

	// Parsing results
	lines := strings.Split(string(data), "\n")
	var results = make([]diskResult, 0)

	for i, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) != 3 {
			return nil, fmt.Errorf("parseDiskResponse: unexpected %d line: %s", i, line)
		}
		disk := strings.TrimSpace(parts[0])

		freeSpace, err := util.ParseNumber(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}

		totalSpace, err := util.ParseNumber(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, err
		}

		results = append(results, diskResult{disk, freeSpace, totalSpace})

	}
	return results, nil
}

func (e *DiskMetricsExporter) Collect(resultLines []diskResult, ch chan<- prometheus.Metric) {
	for _, dm := range resultLines {
		newFreeSpaceMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: e.Namespace,
			Name:      "free_space_in_bytes",
			Help:      "Disks free_space_in_bytes capacity",
		}, []string{"disk"}).WithLabelValues(dm.disk)
		newFreeSpaceMetric.Set(dm.freeSpace)
		newFreeSpaceMetric.Collect(ch)

		newTotalSpaceMetric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: e.Namespace,
			Name:      "total_space_in_bytes",
			Help:      "Disks total_space_in_bytes capacity",
		}, []string{"disk"}).WithLabelValues(dm.disk)
		newTotalSpaceMetric.Set(dm.totalSpace)
		newTotalSpaceMetric.Collect(ch)
	}
}
