package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/TykTechnologies/opentelemetry/trace"
	"github.com/TykTechnologies/tyk/log"
)

var logger = log.Get()

// AddFooBarHeader adds a custom header to the request.
func AddFooBarHeader(rw http.ResponseWriter, r *http.Request) {
	// We create a new span using the context from the incoming request.
	ctx, newSpan := trace.NewSpanFromContext(r.Context(), "", "GoPlugin_first-span")

	// Ensure that the span is properly ended when the function completes.
	defer newSpan.End()

	// Set a new name for the span.
	newSpan.SetName("AddFooBarHeader Testing")

	// Set the status of the span.
	newSpan.SetStatus(trace.SPAN_STATUS_OK, "")

	// Set an attribute on the span.
	newSpan.SetAttributes(trace.NewAttribute("go_plugin", "1"))

	// Call another function, passing in the context (which includes the new span).
	NewFunc(ctx)

	// Add a custom "Foo: Bar" header to the request.
	r.Header.Add("Foo", "Bar")
}

// Enhance the tracing of the plugin execution with additional spans for each function.
// Using Context propagation you can link the spans these span that cover multiple functions.
// Help understand sequence of operations, performance bottlenecks and analyze application behaviour.
func NewFunc(ctx context.Context) {
	// Create a new span using the context passed from the previous function.
	ctx, newSpan := trace.NewSpanFromContext(ctx, "", "GoPlugin_second-span")
	defer newSpan.End()

	// Simulate some processing time.
	time.Sleep(1 * time.Second)

	// Set an attribute on the new span.
	newSpan.SetAttributes(trace.NewAttribute("go_plugin", "2"))

	// Call a function that will record an error and set the span status to error.
	NewFuncWithError(ctx)
}

// Error handling in OpenTelemtry. First create a new span, then set the status to "error",
// then add an attribute. Last use RecordError() to log an error event. This two-step process
// ensures that both the error event is recorded and the span status is set to reflect the error.
func NewFuncWithError(ctx context.Context) {
	// Start a new span using the context passed from the previous function.
	_, newSpan := trace.NewSpanFromContext(ctx, "", "GoPlugin_third-span")
	defer newSpan.End()

	// Set status to error.
	newSpan.SetStatus(trace.SPAN_STATUS_ERROR, "Error Description")

	// Set an attribute on the new span.
	newSpan.SetAttributes(trace.NewAttribute("go_plugin", "3"))

	// Record an error in the span.
	newSpan.RecordError(errors.New("this is an auto-generated error"))
}

func main() {}

func init() {
	logger.Info("--- Go custom plugin v4 init success! ---- ")
}
