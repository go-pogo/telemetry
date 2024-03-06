// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"github.com/go-pogo/env"
	"github.com/go-pogo/env/envtest"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	t.Run("ensure correct config pointers", func(t *testing.T) {
		b := NewBuilder(Config{})
		assert.Same(t, &b.Config.Meter, b.MeterProvider.MeterProviderConfig)
		assert.Same(t, &b.Config.Tracer, b.TracerProvider.TracerProviderConfig)
	})
}

func TestNewDevelopmentBuilder(t *testing.T) {
	b := NewDevelopmentBuilder(Config{})
	assert.Equal(t, alwaysOn, b.Tracer.Sampler)
	assert.Equal(t, []trace.SpanExporter{StdoutSpanExporter()}, b.TracerProvider.SpanExporters)
}

func TestBuilder_Global(t *testing.T) {
	b := NewBuilder(Config{})
	assert.False(t, b.MeterProvider.SetGlobal)
	assert.False(t, b.TracerProvider.SetGlobal)

	b.Global()
	assert.True(t, b.MeterProvider.SetGlobal)
	assert.True(t, b.TracerProvider.SetGlobal)
}

func TestBuilder_Build(t *testing.T) {
	t.Run("ensure envs are loaded", func(t *testing.T) {
		e := envtest.Prepare(nil)
		defer e.Restore()

		_, haveErr := NewBuilder(Config{}).Build()
		assert.NoError(t, haveErr)
		assert.Same(t, env.Environ(), env.Map{
			//"OTEL_RESOURCE_ATTRIBUTES": env.Value(c.ResourceAttributes),
			"OTEL_EXPORTER_OTLP_ENDPOINT":    "",
			"OTEL_EXPORTER_OTLP_HEADERS":     "",
			"OTEL_EXPORTER_OTLP_PROTOCOL":    "grpc",
			"OTEL_EXPORTER_OTLP_TIMEOUT":     "10000",
			"OTEL_EXPORTER_OTLP_CERTIFICATE": "",
		})
	})
}
