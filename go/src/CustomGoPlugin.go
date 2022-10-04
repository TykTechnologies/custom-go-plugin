package main

import (
	"net/http"

	"github.com/TykTechnologies/tyk/log"
)

var logger = log.Get()

// AddFooBarHeader adds custom "Foo: Bar" header to the request
func AddFooBarHeader(rw http.ResponseWriter, r *http.Request) {
	r.Header.Add("Foo", "Bar v4")
}

func main() {}

func init() {
	logger.Info("--- Go custom plugin v4 init success! ---- ")
}
