// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package metricstore

import (
	"net/http"
	"time"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
)

// NewFakeMetricStore return a fake metric store
func NewFakeMetricStore() MetricStore {
	return &fakeMetricStore{}
}

type fakeMetricStore struct{}

type fakeMetricHandler struct {
	metric string
}

func (m *fakeMetricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(m.metric))
}

func (ms *fakeMetricStore) Start() {}

func (ms *fakeMetricStore) Stop() {}

func (ms *fakeMetricStore) Handler() http.Handler {
	return &fakeMetricHandler{metric: "OK"}
}

func (ms *fakeMetricStore) SetUpdateLatencySec(dur time.Duration) {}

func (ms *fakeMetricStore) IncArmAPIUpdateCallFailureCounter() {}

func (ms *fakeMetricStore) IncArmAPIUpdateCallSuccessCounter() {}

func (ms *fakeMetricStore) IncArmAPICallCounter() {}

func (ms *fakeMetricStore) IncK8sAPIEventCounter() {}

func (ms *fakeMetricStore) IncAddressPoolSlowUpdateCounter() {}

func (ms *fakeMetricStore) IncErrorCount(controllererrors.ErrorCode) {}
