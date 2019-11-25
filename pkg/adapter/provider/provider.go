/*
Copyright 2017 The Kubernetes Authors.
Copyright 2019 Riley Berton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"fmt"
	"net/url"
	"time"

	"encoding/json"

	circonus "github.com/circonus-labs/go-apiclient"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog"

	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type clock interface {
	Now() time.Time
}

type realClock struct{}

func (c realClock) Now() time.Time {
	return time.Now()
}

// StackdriverProvider is a provider of custom metrics from Stackdriver.
type CirconusProvider struct {
	kubeClient     *corev1.CoreV1Client
	circonusApiURL string
	apiClients     map[string]*circonus.API
}

// NewCirconusProvider creates a CirconusProvider
func NewCirconusProvider(kubeClient *corev1.CoreV1Client, circonus_api_url string) provider.MetricsProvider {
	return &CirconusProvider{
		kubeClient:     kubeClient,
		circonusApiURL: circonus_api_url,
		apiClients:     make(map[string]*circonus.API, 0),
	}
}

// ListAllMetrics returns all custom metrics available.
// Not implemented (currently returns empty list).
func (p *CirconusProvider) ListAllMetrics() []provider.CustomMetricInfo {
	return []provider.CustomMetricInfo{}
}

// GetMetricByName fetches a particular metric for a particular object.
// The namespace will be empty if the metric is root-scoped.
func (p *CirconusProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	return nil, NewOperationNotSupportedError("GetMetricByName not supported at this time")
}

// GetMetricBySelector fetches a particular metric for a set of objects matching
// the given label selector. The namespace will be empty if the metric is root-scoped.
func (p *CirconusProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	return nil, NewOperationNotSupportedError("GetMetricByName not supported at this time")
}

func CreateURLWithQuery(uri string, param map[string]interface{}) (string, error) {
	urlObj, err := url.Parse(uri)
	if err != nil {
		return uri, err
	}

	query := urlObj.Query()
	for k, v := range param {
		query.Set(k, fmt.Sprintf("%v", v))
	}

	urlObj.RawQuery = query.Encode()
	return urlObj.String(), nil
}

// GetExternalMetric queries Circonus using CAQL to fetch data
// namespace is ignored as well as labels.Selector
func (p *CirconusProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	metricValues := []external_metrics.ExternalMetricValue{}

	caqlQuery := info.Metric

	// get last 5 minutes
	endTime := time.Now()
	startTime := endTime.Add(-(5 * time.Minute))

	param := map[string]interface{}{
		"period": 60,
		"start":  startTime.Unix(),
		"end":    endTime.Unix(),
		"query":  caqlQuery,
	}

	var apiClient *circonus.API = nil

	klog.Infof("Incoming query: %s", caqlQuery)
	klog.Infof("Incoming selector: %s", metricSelector.String())

	// lookup the apiClient using the metric_selector
	reqs, _ := metricSelector.Requirements()

	for _, req := range reqs {
		if req.Key() == "circonus_api_key" {
			var key string
			var ok bool
			if key, ok = req.Values().PopAny(); !ok {
				// no key, return empty set
				return &external_metrics.ExternalMetricValueList{
					Items: metricValues,
				}, nil
			}

			if c, ok := p.apiClients[key]; ok {
				apiClient = c
				break
			}

			apiConfig := &circonus.Config{
				URL:      p.circonusApiURL,
				TokenKey: key,
				TokenApp: "custom-metrics-circonus-adapter",
			}

			apiclient, err := circonus.NewAPI(apiConfig)
			if err != nil {
				return nil, err
			}
			p.apiClients[key] = apiclient
			apiClient = apiclient

		}
	}

	queryString, err := CreateURLWithQuery("/caql", param)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := apiClient.Get(queryString)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	json.Unmarshal(jsonBytes, &result)

	if _, ok := result["_data"]; !ok {
		return nil, apierr.NewInternalError(fmt.Errorf("Circonus response missing _data field"))
	}

	data := result["_data"].([]interface{})
	if len(data) <= 0 {
		// This shouldn't happen with correct query to Circonus
		return nil, apierr.NewInternalError(fmt.Errorf("Empty time series returned from Circonus CAQL query"))
	}

	// point is an array of [time, value1, value2, ..., valueN]
	// we will use, time and value1
	point := data[len(data)-1].([]interface{})
	resultEndTime := point[0].(float64)
	if time.Unix(int64(resultEndTime), 0).After(endTime) {
		return nil, apierr.NewInternalError(fmt.Errorf("Timeseries from Circonus has incorrect end time: %s", resultEndTime))
	}
	metricValue := external_metrics.ExternalMetricValue{
		Timestamp:  metav1.NewTime(time.Unix(int64(resultEndTime), 0)),
		MetricName: caqlQuery,
	}
	value := point[1].(float64)
	metricValue.Value = *resource.NewMilliQuantity(int64(value*1000), resource.DecimalSI)
	metricValues = append(metricValues, metricValue)
	return &external_metrics.ExternalMetricValueList{
		Items: metricValues,
	}, nil
}

// ListAllExternalMetrics returns a list of available external metrics.
// Not implemented (currently returns empty list).
func (p *CirconusProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	return []provider.ExternalMetricInfo{}
}
