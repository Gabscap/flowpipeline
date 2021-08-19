package segments

import (
	"log"
	"os"
	"sync"

	"github.com/bwNetFlow/consumer_prometheus/exporter"
)

type PrometheusExporter struct {
	BaseSegment
	PromEndpoint string
}

func (segment PrometheusExporter) New(config map[string]string) Segment {
	return &PrometheusExporter{
		PromEndpoint: config["endpoint"],
	}
}

func (segment *PrometheusExporter) Run(wg *sync.WaitGroup) {
	if segment.PromEndpoint == "" {
		log.Println("[error] PrometheusExporter: Missing required configuration parameter.")
		os.Exit(1)
	}

	defer func() {
		close(segment.out)
		wg.Done()
	}()

	var promExporter = exporter.Exporter{}
	promExporter.Initialize()
	promExporter.ServeEndpoints(segment.PromEndpoint)

	for msg := range segment.in {
		segment.out <- msg
		promExporter.Increment(msg)
	}
}

func init() {
	segment := &PrometheusExporter{}
	RegisterSegment("prometheusexporter", segment)
}