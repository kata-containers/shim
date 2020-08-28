// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"bytes"
	"context"
	"io"
	"runtime/pprof"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

// Implements jaeger-client-go.Logger interface
type traceLogger struct {
}

// tracerCloser contains a copy of the closer returned by createTracer() which
// is used by stopTracing().
var tracerCloser io.Closer

func (t traceLogger) Error(msg string) {
	shimLog.Error(msg)
}

func (t traceLogger) Infof(msg string, args ...interface{}) {
	shimLog.Infof(msg, args...)
}

func createTracer(name string) (opentracing.Tracer, error) {
	cfg := &config.Configuration{
		ServiceName: name,

		// If tracing is disabled, use a NOP trace implementation
		Disabled: !tracing,

		// Note that span logging reporter option cannot be enabled as
		// it pollutes the output stream which causes (atleast) the
		// "state" command to fail under Docker.
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},

		// Ensure that Jaeger logs each span.
		// This is essential as it is used by:
		//
		// https: //github.com/kata-containers/tests/blob/master/tracing/tracing-test.sh
		Reporter: &config.ReporterConfig{
			LogSpans: tracing,
		},
	}

	logger := traceLogger{}

	tracer, closer, err := cfg.NewTracer(config.Logger(logger))
	if err != nil {
		return nil, err
	}

	// save for stopTracing()'s exclusive use
	tracerCloser = closer

	// Seems to be essential to ensure non-root spans are logged
	opentracing.SetGlobalTracer(tracer)

	return tracer, nil
}

// stopTracing() ends all tracing, reporting the spans to the collector.
func stopTracing(ctx context.Context) {
	shimLog.Infof("FIXME: stopTracing: tracing: %+v", tracing)

	if !tracing {
		return
	}

	shimLog.Infof("FIXME: stopTracing: calling opentracing.SpanFromContext")
	span := opentracing.SpanFromContext(ctx)
	shimLog.Infof("FIXME: stopTracing: opentracing.SpanFromContext returned span %+v", span)
	if span != nil {
		shimLog.Infof("FIXME: stopTracing: finishing span")
		span.Finish()
	}

	shimLog.Infof("FIXME: stopTracing: closing tracer")

	// report all possible spans to the collector
	tracerCloser.Close()

	shimLog.Infof("FIXME: stopTracing: DONE")
}

// trace creates a new tracing span based on the specified name and parent
// context.
func trace(parent context.Context, name string) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(parent, name)

	span.SetTag("source", "shim")

	// This is slightly confusing: when tracing is disabled, trace spans
	// are still created - but the tracer used is a NOP. Therefore, only
	// display the message when tracing is really enabled.
	if tracing {
		bt := getBacktrace()
		// This log message is *essential*: it is used by:
		// https: //github.com/kata-containers/tests/blob/master/tracing/tracing-test.sh
		shimLog.Debugf("created span %v (name: %q, backtrace: %v)", span, name, bt)
	}

	return span, ctx
}

func getBacktrace() string {
	profiles := pprof.Profiles()

	buf := &bytes.Buffer{}

	for _, p := range profiles {
		// The magic number requests a full stacktrace. See
		// https://golang.org/pkg/runtime/pprof/#Profile.WriteTo.
		pprof.Lookup(p.Name()).WriteTo(buf, 2)
	}

	//return buf.String()
	return strings.Join(strings.Split(buf.String(), "\n"), ",")
}
