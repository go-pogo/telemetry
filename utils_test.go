// Copyright (c) 2024, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package telemetry

import (
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"testing"
)

func TestAttributesFromMap(t *testing.T) {
	want := []attribute.KeyValue{
		attribute.String("key1", "value1"),
		attribute.String("key2", "value2"),
	}

	assert.Equal(t, want, AttributesFromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	}))
}

func TestStdoutSpanExporter(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = StdoutSpanExporter()
	})
}
