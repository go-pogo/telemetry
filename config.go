// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"github.com/go-pogo/env"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/rawconv"
)

var _ env.Environment = (*Config)(nil)

type Config struct {
	Meter  MeterProviderConfig
	Tracer TracerProviderConfig

	// https://opentelemetry.io/docs/languages/sdk-configuration/general/
	//ResourceAttributes      map[string]string `env:"OTEL_RESOURCE_ATTRIBUTES"`
	ExporterOTLPEndpoint    string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	ExporterOTLPHeaders     string `env:"OTEL_EXPORTER_OTLP_HEADERS"`
	ExporterOTLPProtocol    string `env:"OTEL_EXPORTER_OTLP_PROTOCOL" default:"grpc"`
	ExporterOTLPTimeout     uint64 `env:"OTEL_EXPORTER_OTLP_TIMEOUT" default:"10000"` // 10s
	ExporterOTLPCertificate string `env:"OTEL_EXPORTER_OTLP_CERTIFICATE"`
}

func (c Config) Environ() (env.Map, error) {
	return env.Map{
		//"OTEL_RESOURCE_ATTRIBUTES": env.Value(c.ResourceAttributes),
		"OTEL_EXPORTER_OTLP_ENDPOINT":    env.Value(c.ExporterOTLPEndpoint),
		"OTEL_EXPORTER_OTLP_HEADERS":     env.Value(c.ExporterOTLPHeaders),
		"OTEL_EXPORTER_OTLP_PROTOCOL":    env.Value(c.ExporterOTLPProtocol),
		"OTEL_EXPORTER_OTLP_TIMEOUT":     rawconv.ValueFromUint64(c.ExporterOTLPTimeout),
		"OTEL_EXPORTER_OTLP_CERTIFICATE": env.Value(c.ExporterOTLPCertificate),
	}, nil
}

type Builder struct {
	Config
	MeterProvider  *MeterProviderBuilder
	TracerProvider *TracerProviderBuilder
}

func NewBuilder(c Config) *Builder {
	b := Builder{Config: c}

	// make sure the configs are referenced from the copy of c,
	// which is set to Builder.Config
	b.MeterProvider = NewMeterProviderBuilder(&b.Config.Meter)
	b.TracerProvider = NewTracerProviderBuilder(&b.Config.Tracer)
	return &b
}

func NewDevelopmentBuilder(c Config) *Builder {
	b := NewBuilder(c)
	b.Tracer.Sampler = "always_on"
	//b.TracerProvider.WithSpanExporters(StdoutSpanExporter())
	return b
}

func (b *Builder) Global() *Builder {
	if b.MeterProvider != nil {
		b.MeterProvider.SetGlobal = true
	}
	if b.TracerProvider != nil {
		b.TracerProvider.SetGlobal = true
	}
	return b
}

func (b *Builder) WithDefaultExporter() *Builder {
	switch b.ExporterOTLPProtocol {
	case "grpc":
		b.MeterProvider.WithGrpcExporter()
		b.TracerProvider.WithGrpcExporter()
	}
	return b
}

func (b *Builder) Build() (*Telemetry, error) {
	if err := env.Load(b.Config); err != nil {
		return nil, err
	}

	m, err1 := b.MeterProvider.Build()
	t, err2 := b.TracerProvider.Build()

	if err1 != nil || err2 != nil {
		return nil, errors.Append(err1, err2)
	}

	return New(m, t), nil
}

type errList struct {
	errs []error
}

func (e *errList) append(errs ...error) {
	if e.errs == nil {
		e.errs = make([]error, 0, 2)
	}
	e.errs = append(e.errs, errs...)
}

func (e *errList) join() error {
	err := errors.Join(e.errs...)
	if err != nil {
		e.errs = e.errs[:]
	}
	return err
}
