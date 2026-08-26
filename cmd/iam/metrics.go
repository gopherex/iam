package main

// Prometheus scrape endpoint.
//
// The service exports OTLP, which is the right default for a collector-based
// stack and useless to the many deployments whose monitoring pulls rather than
// receives. Both can be true at once: OTel metrics are produced once and read by
// as many readers as are attached, so adding a Prometheus reader costs a second
// reader on the same provider — not a second set of instruments.
//
// Taking the meter provider over from the telemetry SDK is what makes that
// possible, so this is only done when scraping is on; otherwise the SDK's own
// pipeline is left exactly as it was.

import (
	"context"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	metricexport "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// metricsPath is where the scrape endpoint is served, on the probe listener —
// the same port a cluster already scrapes and does not expose publicly.
const metricsPath = "/metrics"

// otlpMetricsConfigured reports whether the standard OTel environment names an
// endpoint for metrics. Without one there is nothing to push to, and the
// Prometheus reader is the whole pipeline.
func otlpMetricsConfigured() bool {
	for _, key := range []string{
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
	} {
		if os.Getenv(key) != "" {
			return true
		}
	}

	return false
}

// setupMetrics installs a meter provider that feeds a Prometheus registry, and
// the OTLP exporter as well when one is configured. It returns the scrape
// handler and a shutdown.
func setupMetrics(
	ctx context.Context, serviceName, version, instanceID string,
) (http.Handler, func(context.Context) error, error) {
	registry := prometheus.NewRegistry()

	exporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		return nil, nil, err
	}

	// Built without merging resource.Default(): its schema URL tracks the SDK
	// release and merging across two schema versions is an error, not a warning.
	// The three attributes below are the ones a scrape needs to tell instances
	// apart.
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(version),
		semconv.ServiceInstanceID(instanceID),
	)

	opts := []sdkmetric.Option{
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	}

	if otlpMetricsConfigured() {
		otlp, oerr := metricexport.New(ctx)
		if oerr != nil {
			return nil, nil, oerr
		}

		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(otlp)))
	}

	provider := sdkmetric.NewMeterProvider(opts...)
	otel.SetMeterProvider(provider)

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{}), provider.Shutdown, nil
}
