// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	"os"
)

func AttributesFromMap(m map[string]string) []attribute.KeyValue {
	res := make([]attribute.KeyValue, 0, len(m))
	for k, v := range m {
		res = append(res, attribute.String(k, v))
	}
	return res
}

func StdoutSpanExporter() trace.SpanExporter {
	exp, err := stdouttrace.New(
		stdouttrace.WithWriter(os.Stdout),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		// err should always be nil
		panic(fmt.Sprintf("telemetry.StdoutSpanExporter: %+v", err))
	}
	return exp
}
