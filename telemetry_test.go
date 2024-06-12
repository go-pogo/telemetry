// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestTelemetry_MeterProvider(t *testing.T) {
	t.Run("noop", func(t *testing.T) {
		assert.IsType(t, metricnoop.MeterProvider{}, new(Telemetry).MeterProvider())
	})
	t.Run("same", func(t *testing.T) {
		prov := metric.NewMeterProvider()
		assert.Same(t, prov, New(prov, nil).MeterProvider())
	})
}

func TestTelemetry_TracerProvider(t *testing.T) {
	t.Run("noop", func(t *testing.T) {
		assert.IsType(t, tracenoop.TracerProvider{}, new(Telemetry).TracerProvider())
	})
	t.Run("same", func(t *testing.T) {
		prov := trace.NewTracerProvider()
		assert.Same(t, prov, New(nil, prov).TracerProvider())
	})
}
