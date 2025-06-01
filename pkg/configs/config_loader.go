package configs

import (
	"flag"
	"os"
)

type Configuration struct {
	ListeningAddress    *string
	MetricsEndpoint     *string
	ClickhouseOnly      *bool
	Insecure            *bool
	ClickhouseScrapeURI string
	User                string
	Password            string
}

func LoadConfigs() Configuration {
	configs := Configuration{
		ListeningAddress:    flag.String("telemetry.address", ":9116", "Address on which to expose metrics."),
		MetricsEndpoint:     flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics."),
		ClickhouseOnly:      flag.Bool("clickhouse_only", false, "Expose only Clickhouse metrics, not metrics from the exporter itself"),
		Insecure:            flag.Bool("insecure", true, "Ignore server certificate if using https"),
		ClickhouseScrapeURI: os.Getenv("CLICKHOUSE_URI"),
		User:                os.Getenv("CLICKHOUSE_USER"),
		Password:            os.Getenv("CLICKHOUSE_PASSWORD"),
	}

	// must be called after all flags are defined and before flags are accessed by the program
	flag.Parse()

	return configs
}
