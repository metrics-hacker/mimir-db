// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/pkg/querier/queryrange/instrumentation.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.

package querymiddleware

import (
	"context"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/weaveworks/common/instrument"
)

// newInstrumentMiddleware can be inserted into the middleware chain to expose timing information.
func newInstrumentMiddleware(name string, metrics *instrumentMiddlewareMetrics) Middleware {
	var durationCol instrument.Collector

	// Support the case metrics shouldn't be tracked (ie. unit tests).
	if metrics != nil {
		durationCol = instrument.NewHistogramCollector(metrics.duration)
	} else {
		durationCol = &noopCollector{}
	}

	return MiddlewareFunc(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			var resp Response
			err := instrument.CollectedRequest(ctx, name, durationCol, instrument.ErrorCode, func(ctx context.Context) error {
				sp := opentracing.SpanFromContext(ctx)
				if sp != nil {
					req.LogToSpan(sp)
				}

				var err error
				resp, err = next.Do(ctx, req)
				return err
			})
			return resp, err
		})
	})
}

// instrumentMiddlewareMetrics holds the metrics tracked by newInstrumentMiddleware.
type instrumentMiddlewareMetrics struct {
	duration *prometheus.HistogramVec
}

// newInstrumentMiddlewareMetrics makes a new instrumentMiddlewareMetrics.
func newInstrumentMiddlewareMetrics(registerer prometheus.Registerer) *instrumentMiddlewareMetrics {
	return &instrumentMiddlewareMetrics{
		duration: promauto.With(registerer).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "cortex",
			Name:      "frontend_query_range_duration_seconds",
			Help:      "Total time spent in seconds doing query range requests.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "status_code"}),
	}
}

// noopCollector is a noop collector that can be used as placeholder when no metric
// should tracked by the instrumentation.
type noopCollector struct{}

// Register implements instrument.Collector.
func (c *noopCollector) Register() {}

// Before implements instrument.Collector.
func (c *noopCollector) Before(context.Context, string, time.Time) {}

// After implements instrument.Collector.
func (c *noopCollector) After(context.Context, string, string, time.Time) {}
