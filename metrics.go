// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"time"

	"github.com/go-pogo/env"
	"github.com/go-pogo/errors"
	"github.com/go-pogo/rawconv"
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"golang.org/x/net/context"
)

var _ env.Environment = (*MeterProviderConfig)(nil)

// MeterProviderConfig holds the configuration for the MeterProviderBuilder.
// These values may be set using any external source, such as environment
// variables, config files, etc.
type MeterProviderConfig struct {
	Enabled bool `env:"OTEL_METRIC_ENABLED" default:"true"`

	// OTEL_EXPORTER_OTLP_METRICS_ENDPOINT
	// OTEL_EXPORTER_OTLP_METRICS_HEADERS
	// OTEL_EXPORTER_OTLP_METRICS_TIMEOUT

	// ExportInterval is the time interval (in milliseconds) between the start
	// of two export attempts.
	ExportInterval int `env:"OTEL_METRIC_EXPORT_INTERVAL" default:"60000"`
	// ExportTimeout is the maximum allowed time (in milliseconds) to export
	// data.
	ExportTimeout int `env:"OTEL_METRIC_EXPORT_TIMEOUT" default:"30000"`
}

func (c MeterProviderConfig) ExportIntervalDuration() time.Duration {
	return time.Duration(c.ExportInterval) * time.Millisecond
}

func (c MeterProviderConfig) ExportTimeoutDuration() time.Duration {
	return time.Duration(c.ExportTimeout) * time.Millisecond

}

func (c MeterProviderConfig) Environ() (env.Map, error) {
	return env.Map{
		"OTEL_METRIC_EXPORT_INTERVAL": rawconv.ValueFromInt(c.ExportInterval),
		"OTEL_METRIC_EXPORT_TIMEOUT":  rawconv.ValueFromInt(c.ExportTimeout),
	}, nil
}

type MeterProviderBuilder struct {
	*MeterProviderConfig

	opts []metric.Option
	errs errList

	DisableRuntimeMetrics bool
	SetGlobal             bool
}

func NewMeterProviderBuilder(conf *MeterProviderConfig) *MeterProviderBuilder {
	if conf == nil {
		conf = new(MeterProviderConfig)
	}
	return &MeterProviderBuilder{MeterProviderConfig: conf}
}

func (met *MeterProviderBuilder) With(opts ...metric.Option) *MeterProviderBuilder {
	if met.opts == nil {
		met.opts = opts
		return met
	}

	met.opts = append(met.opts, opts...)
	return met
}

func (met *MeterProviderBuilder) WithReader(r metric.Reader, views ...metric.View) *MeterProviderBuilder {
	met.With(metric.WithReader(r))
	if len(views) > 0 {
		met.With(metric.WithView(views...))
	}
	return met
}

func (met *MeterProviderBuilder) WithGrpcExporter(opts ...otlpmetricgrpc.Option) *MeterProviderBuilder {
	exp, err := otlpmetricgrpc.New(context.Background(), opts...)
	if err != nil {
		met.errs.append(errors.WithStack(err))
		return met
	}

	return met.With(metric.WithReader(metric.NewPeriodicReader(exp)))
}

func (met *MeterProviderBuilder) WithPrometheusExporter(reg prometheus.Registerer, views ...metric.View) *MeterProviderBuilder {
	if reg == nil {
		return met
	}

	exp, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		met.errs.append(errors.WithStack(err))
		return met
	}

	return met.WithReader(exp, views...)
}

func (met *MeterProviderBuilder) Build(opts ...metric.Option) (*metric.MeterProvider, error) {
	if err := met.errs.join(); err != nil {
		return nil, err
	}
	if met.MeterProviderConfig != nil {
		if err := env.Load(met.MeterProviderConfig); err != nil {
			return nil, err
		}
	}

	opts = append(met.opts, opts...)
	prov := metric.NewMeterProvider(opts...)

	if !met.DisableRuntimeMetrics {
		if err := runtimemetrics.Start(runtimemetrics.WithMeterProvider(prov)); err != nil {
			return prov, errors.WithStack(err)
		}
	}
	if met.SetGlobal {
		otel.SetMeterProvider(prov)
	}
	return prov, nil
}
