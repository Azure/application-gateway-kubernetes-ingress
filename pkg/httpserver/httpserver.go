// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controller"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/health"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
)

// HTTPServer serving probes and metrics
type HTTPServer interface {
	Start()
	Stop()
}

type httpServer struct {
	server *http.Server
}

// NewHealthMux makes a new *http.ServeMux
func NewHealthMux(handlers map[string]http.Handler) *http.ServeMux {
	router := http.NewServeMux()
	for url, handler := range handlers {
		router.Handle(url, handler)
	}

	return router
}

// NewHTTPServer creates a new api server
func NewHTTPServer(controller *controller.AppGwIngressController, metricStore metricstore.MetricStore, apiPort string) HTTPServer {
	return &httpServer{
		server: &http.Server{
			Addr: fmt.Sprintf(":%s", apiPort),
			Handler: NewHealthMux(map[string]http.Handler{
				"/health/ready": health.ReadinessHandler(controller),
				"/health/alive": health.LivenessHandler(controller),
				"/metrics":      metricStore.Handler(),
			}),
		},
	}
}

func (s *httpServer) Start() {
	go func() {
		klog.Infof("Starting API Server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil {
			klog.Fatal("Failed to start API server", err)
		}
	}()
}

func (s *httpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		klog.Error("Unable to shutdown API server gracefully", err)
	}
}
