package main

import (
	"net/http"

	"github.com/TykTechnologies/opentelemetry/trace"
	"github.com/TykTechnologies/tyk/ctx"

	"github.com/TykTechnologies/tyk/log"
	"github.com/TykTechnologies/tyk/user"
)

var logger = log.Get()

// AddFooBarHeader adds custom "Foo: Bar" header to the request
func AddFooBarHeader(rw http.ResponseWriter, r *http.Request) {
	// We create a new span using the context from the incoming request.
	_, newSpan := trace.NewSpanFromContext(r.Context(), "", "GoPlugin_first-span")

	// Ensure that the span is properly ended when the function completes.
	defer newSpan.End()

	// Set a new name for the span.
	newSpan.SetName("AddFooBarHeader Function")

	// Set the status of the span.
	newSpan.SetStatus(trace.SPAN_STATUS_OK, "")

	r.Header.Add("Foo", "Bar")
}

// Custom Auth, applies a rate limit of
// 2 per 10 given a token of "abc"
func AuthCheck(rw http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")

	_, newSpan := trace.NewSpanFromContext(r.Context(), "", "GoPlugin_custom-auth")
	defer newSpan.End()

	if token != "d3fd1a57-94ce-4a36-9dfe-679a8f493b49" && token != "3be61aa4-2490-4637-93b9-105001aa88a5" {
		newSpan.SetAttributes(trace.NewAttribute("auth", "failed"))
		newSpan.SetStatus(trace.SPAN_STATUS_ERROR, "")

		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	newSpan.SetAttributes(trace.NewAttribute("auth", "success"))
	newSpan.SetStatus(trace.SPAN_STATUS_OK, "")

	session := &user.SessionState{
		Alias: token,
		Rate:  2,
		Per:   10,
		MetaData: map[string]interface{}{
			token: token,
		},
		KeyID: token,
	}

	ctx.SetSession(r, session, true)
}

func main() {}

func init() {
	logger.Info("--- Go custom plugin v4 init success! ---- ")
}
