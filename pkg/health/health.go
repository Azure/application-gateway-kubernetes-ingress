// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package health

import "net/http"

type Probe func() bool

type HealthProbes interface {
	Liveness() bool
	Readiness() bool
}

func makeHandler(router *http.ServeMux, url string, probe Probe) {
	router.Handle(url, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(map[bool]int{
			true:  http.StatusOK,
			false: http.StatusServiceUnavailable,
		}[probe()])
	}))
}

// NewHealthMux makes a new *http.ServeMux
func NewHealthMux(probes HealthProbes) *http.ServeMux {
	router := http.NewServeMux()
	var handlers = map[string]Probe{
		"/health/ready": probes.Readiness,
		"/health/alive": probes.Liveness,
	}
	for url, probe := range handlers {
		makeHandler(router, url, probe)
	}
	return router
}
