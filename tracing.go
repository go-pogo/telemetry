// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"github.com/go-pogo/env"
	"github.com/go-pogo/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"golang.org/x/net/context"
	"runtime/debug"
	"strings"
)

var _ env.Environment = (*TracerProviderConfig)(nil)

type TracerProviderConfig struct {
	Enabled bool `env:"OTEL_TRACES_ENABLED" default:"true"`

	// OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
	// OTEL_EXPORTER_OTLP_TRACES_HEADERS
	// OTEL_EXPORTER_OTLP_TRACES_TIMEOUT

	Sampler    string `env:"OTEL_TRACES_SAMPLER,noprefix" default:"parentbased_traceidratio"`
	SamplerArg string `env:"OTEL_TRACES_SAMPLER_ARG,noprefix" default:"0.5"`
}

func (c TracerProviderConfig) Environ() (env.Map, error) {
	return env.Map{
		"OTEL_TRACES_SAMPLER":     env.Value(c.Sampler),
		"OTEL_TRACES_SAMPLER_ARG": env.Value(c.SamplerArg),
	}, nil
}

type TracerProviderBuilder struct {
	*TracerProviderConfig

	opts []trace.TracerProviderOption
	errs errList

	SetGlobal     bool
	Attributes    []attribute.KeyValue
	Sampler       trace.Sampler
	SpanExporters []trace.SpanExporter
}

func NewTracerProviderBuilder(conf *TracerProviderConfig) *TracerProviderBuilder {
	if conf == nil {
		conf = new(TracerProviderConfig)
	}
	return &TracerProviderBuilder{TracerProviderConfig: conf}
}

func (tra *TracerProviderBuilder) With(opts ...trace.TracerProviderOption) *TracerProviderBuilder {
	if tra.opts == nil {
		tra.opts = opts
		return tra
	}

	tra.opts = append(tra.opts, opts...)
	return tra
}

func (tra *TracerProviderBuilder) WithSampler(sample trace.Sampler) *TracerProviderBuilder {
	tra.Sampler = sample
	return tra
}

func (tra *TracerProviderBuilder) WithAttributes(attrs ...attribute.KeyValue) *TracerProviderBuilder {
	if tra.Attributes == nil {
		tra.Attributes = attrs
		return tra
	}

	tra.Attributes = append(tra.Attributes, attrs...)
	return tra
}

func (tra *TracerProviderBuilder) WithSpanExporters(exporters ...trace.SpanExporter) *TracerProviderBuilder {
	if tra.SpanExporters == nil {
		tra.SpanExporters = exporters
		return tra
	}

	tra.SpanExporters = append(tra.SpanExporters, exporters...)
	return tra
}

func (tra *TracerProviderBuilder) WithGrpcExporter(opts ...otlptracegrpc.Option) *TracerProviderBuilder {
	exp, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		tra.errs.append(errors.WithStack(err))
		return tra
	}

	return tra.With(trace.WithBatcher(exp))
}

func (tra *TracerProviderBuilder) WithBuildInfo(info *debug.BuildInfo, modules ...string) *TracerProviderBuilder {
	if info == nil {
		return tra
	}

	_ = tra.WithAttributes(semconv.ServiceVersion(info.Main.Version))
	for _, set := range info.Settings {
		if !strings.HasPrefix(set.Key, "vcs.") {
			continue
		}

		tra.Attributes = append(tra.Attributes, attribute.String(set.Key, set.Value))
	}
	if len(modules) > 0 {
		for _, dep := range info.Deps {
			for _, name := range modules {
				if dep.Path != name {
					continue
				}

				tra.Attributes = append(tra.Attributes, attribute.String(dep.Path, dep.Version))
			}
		}
	}

	return tra
}

func (tra *TracerProviderBuilder) Build(opts ...trace.TracerProviderOption) (*trace.TracerProvider, error) {
	var err error
	if err = tra.errs.join(); err != nil {
		return nil, err
	}
	if tra.TracerProviderConfig != nil {
		if err = env.Load(tra.TracerProviderConfig); err != nil {
			return nil, err
		}
	}

	opts = append(tra.opts, opts...)
	if tra.Sampler != nil {
		opts = append(opts, trace.WithSampler(tra.Sampler))
	}
	for _, x := range tra.SpanExporters {
		opts = append(opts, trace.WithBatcher(x))
	}

	res := resource.Default()
	if len(tra.Attributes) != 0 {
		res, err = resource.Merge(res, resource.NewSchemaless(tra.Attributes...))
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	opts = append(opts, trace.WithResource(res))
	prov := trace.NewTracerProvider(opts...)

	if tra.SetGlobal {
		otel.SetTracerProvider(prov)
	}
	return prov, nil
}
