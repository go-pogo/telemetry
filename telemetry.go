// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"context"
	"github.com/go-pogo/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"net/http"
)

type MeterProvider = metric.MeterProvider
type TracerProvider = trace.TracerProvider

type Provider interface {
	MeterProvider() MeterProvider
	TracerProvider() TracerProvider
}

type Telemetry struct {
	meter  *metricsdk.MeterProvider
	tracer *tracesdk.TracerProvider
}

func New(m *metricsdk.MeterProvider, t *tracesdk.TracerProvider) *Telemetry {
	return &Telemetry{
		meter:  m,
		tracer: t,
	}
}

// MeterProvider returns the configured MeterProvider or a noop MeterProvider
// when none is configured.
func (t *Telemetry) MeterProvider() MeterProvider {
	if t == nil || t.meter == nil {
		return metricnoop.NewMeterProvider()
	}
	return t.meter
}

// TracerProvider returns the configured TracerProvider or a noop TracerProvider
// when none is configured.
func (t *Telemetry) TracerProvider() TracerProvider {
	if t == nil || t.tracer == nil {
		return tracenoop.NewTracerProvider()
	}
	return t.tracer
}

// NewHttpHandler creates a http.Handler which records metrics for any incoming
// requests handled by h.
func (t *Telemetry) NewHttpHandler(h http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	if len(opts) == 0 {
		opts = append(opts, otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents))
	}
	opts = append(opts,
		otelhttp.WithTracerProvider(t.TracerProvider()),
		otelhttp.WithMeterProvider(t.MeterProvider()),
	)

	return otelhttp.NewHandler(h, operation, opts...)
}

// ForceFlush flushes all pending telemetry and/or immediately exports all spans
// that have not yet been exported for all the registered span processors,
// depending on whether a MeterProvider and/or TracerProvider is configured.
// See MeterProvider.ForceFlush and TracerProvider.ForceFlush for more details.
func (t *Telemetry) ForceFlush(ctx context.Context) error {
	if t == nil {
		return nil
	}

	var err error
	if t.meter != nil {
		err = errors.Append(err, errors.WithStack(t.meter.ForceFlush(ctx)))
	}
	if t.tracer != nil {
		err = errors.Append(err, errors.WithStack(t.tracer.ForceFlush(ctx)))
	}
	return err
}

// Shutdown shuts down the MeterProvider and/or TracerProvider depending on
// whether they are configured.
// See MeterProvider.Shutdown and TracerProvider.Shutdown for more details.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil {
		return nil
	}

	var err error
	if t.meter != nil {
		err = errors.Append(err, errors.WithStack(t.meter.Shutdown(ctx)))
	}
	if t.tracer != nil {
		err = errors.Append(err, errors.WithStack(t.tracer.Shutdown(ctx)))
	}
	return err
}
