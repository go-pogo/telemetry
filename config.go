// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"time"

	"github.com/go-pogo/env"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/rawconv"
)

var _ env.Environment = (*Config)(nil)

type Config struct {
	Meter  MeterProviderConfig
	Tracer TracerProviderConfig

	// https://opentelemetry.io/docs/languages/sdk-configuration/general/
	ServiceName        string            `env:"OTEL_SERVICE_NAME"`
	ResourceAttributes map[string]string `env:"OTEL_RESOURCE_ATTRIBUTES"`

	ExporterOTLP ExporterOTLPConfig
}

var _ env.Environment = (*ExporterOTLPConfig)(nil)

type ExporterOTLPConfig struct {
	Endpoint          string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	Headers           string `env:"OTEL_EXPORTER_OTLP_HEADERS"`
	Protocol          string `env:"OTEL_EXPORTER_OTLP_PROTOCOL" default:"grpc"`
	Timeout           uint64 `env:"OTEL_EXPORTER_OTLP_TIMEOUT" default:"10000"` // 10s
	Certificate       string `env:"OTEL_EXPORTER_OTLP_CERTIFICATE"`
	ClientKey         string `env:"OTEL_EXPORTER_OTLP_CLIENT_KEY"`
	ClientCertificate string `env:"OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE"`
}

func (c ExporterOTLPConfig) TimeoutDuration() time.Duration {
	return time.Duration(c.Timeout) * time.Millisecond
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

const alwaysOn = "always_on"

func NewDevelopmentBuilder(c Config) *Builder {
	b := NewBuilder(c)
	b.Tracer.Sampler = alwaysOn
	b.TracerProvider.WithSpanExporters(StdoutSpanExporter())
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
	if b.ExporterOTLP.Endpoint == "" {
		return b
	}

	switch b.ExporterOTLP.Protocol {
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

func (c Config) Environ() (env.Map, error) {
	attr, err := rawconv.Marshal(c.ResourceAttributes)
	if err != nil {
		return nil, err
	}

	m := env.Map{
		"OTEL_SERVICE_NAME":        env.Value(c.ServiceName),
		"OTEL_RESOURCE_ATTRIBUTES": attr,
	}
	if otlp, _ := c.ExporterOTLP.Environ(); otlp != nil {
		m.MergeValues(otlp)
	}
	return m, nil
}

func (c ExporterOTLPConfig) Environ() (env.Map, error) {
	m := make(env.Map, 5)
	m["OTEL_EXPORTER_OTLP_PROTOCOL"] = env.Value(c.Protocol)
	m["OTEL_EXPORTER_OTLP_TIMEOUT"] = rawconv.ValueFromUint64(c.Timeout)

	if c.Endpoint != "" {
		m["OTEL_EXPORTER_OTLP_ENDPOINT"] = env.Value(c.Endpoint)
	}
	if c.Headers != "" {
		m["OTEL_EXPORTER_OTLP_HEADERS"] = env.Value(c.Headers)
	}
	if c.Certificate != "" {
		m["OTEL_EXPORTER_OTLP_CERTIFICATE"] = env.Value(c.Certificate)
	}
	if c.ClientKey != "" {
		m["OTEL_EXPORTER_OTLP_CLIENT_KEY"] = env.Value(c.ClientKey)
	}
	if c.ClientCertificate != "" {
		m["OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE"] = env.Value(c.ClientCertificate)
	}

	return m, nil
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
