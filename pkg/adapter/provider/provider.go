// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package provider

import (
	"fmt"
	"net/url"
	"time"

	"encoding/json"

	circonus "github.com/circonus-labs/go-apiclient"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"

	kcorev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"

	yaml "gopkg.in/yaml.v2"
)

// type clock interface {
// 	Now() time.Time
// }

type realClock struct{} //nolint:unused

func (c realClock) Now() time.Time {
	return time.Now()
}

// CirconusProvider is a provider of custom metrics from Circonus CAQL.
type CirconusProvider struct {
	// kubeClient     *corev1.CoreV1Client
	circonusAPIURL string
	queryMap       map[string]Query
	apiClients     map[string]*circonus.API
	aggFuncs       map[string]bool
	configChanges  map[string]string
}

// FromYAML loads the configuration from a blob of YAML.
func FromYAML(contents []byte) (*AdapterConfig, error) {
	var cfg AdapterConfig
	if err := yaml.UnmarshalStrict(contents, &cfg); err != nil {
		return nil, fmt.Errorf("unable to parse query config: %v", err)
	}
	return &cfg, nil
}

func ReadConfigMap(provider *CirconusProvider, cm kcorev1.ConfigMap, field string) error {
	config := cm.Data[field]
	if config == "" {
		return fmt.Errorf("cannot load config field: %s", field)
	}
	cfg, err := FromYAML([]byte(config))
	if err != nil {
		return err
	}

	// go through the cfg and make the external name -> Query map.
	// also checks the config for uniqueness of external_name
	for _, q := range cfg.Queries {
		var agg string
		if provider.aggFuncs[q.Aggregate] {
			agg = q.Aggregate
		} else {
			agg = "average"
		}
		q.Aggregate = agg
		provider.queryMap[cm.Namespace+"/"+q.ExternalName] = q
	}
	return nil
}

func CheckConfigMaps(kubeClient *corev1.CoreV1Client, provider *CirconusProvider) error { //nolint:interfacer
	for {
		klog.Infof("Checking config maps for special annotation at: %s", time.Now().UTC())
		if list, err := kubeClient.ConfigMaps("").List(metav1.ListOptions{}); err == nil && list != nil {
			klog.Infof("Found %d config maps in the cluster", len(list.Items))
			for _, cm := range list.Items {
				srv := provider.configChanges[cm.Namespace+"/"+cm.Name]
				rv := cm.ResourceVersion
				if (srv != rv) && len(cm.Annotations) > 0 {
					klog.Infof("Check ConfigMap: %s/%s for annotation", cm.Namespace, cm.Name)
					if x := cm.Annotations["circonus.com/k8s_custom_metrics_config"]; x != "" {
						klog.Infof("Found config map with required annotation, config field: %s", x)
						if err := ReadConfigMap(provider, cm, x); err != nil {
							klog.Errorf("Error reading the config map: %v", err)
						}
					}
					provider.configChanges[cm.Namespace+"/"+cm.Name] = rv
				}
			}
		}
		time.Sleep(10 * time.Second)
	}
	// return nil
}

// NewCirconusProvider creates a CirconusProvider
func NewCirconusProvider(kubeClient *corev1.CoreV1Client, circonusAPIURL string, configFile string) provider.MetricsProvider {

	provider := CirconusProvider{}
	provider.queryMap = make(map[string]Query)
	provider.apiClients = make(map[string]*circonus.API)
	provider.circonusAPIURL = circonusAPIURL
	provider.configChanges = make(map[string]string)
	provider.aggFuncs = make(map[string]bool, 3)
	provider.aggFuncs["average"] = true
	provider.aggFuncs["min"] = true
	provider.aggFuncs["max"] = true

	go func() {
		_ = CheckConfigMaps(kubeClient, &provider)
	}()
	return &provider
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
	return nil, NewOperationNotSupportedError("GetMetricBySelector not supported at this time")
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

	// get the query from the configMap
	var query Query
	ok := false
	if query, ok = p.queryMap[namespace+"/"+info.Metric]; !ok {
		// no matching query, return empty set
		return &external_metrics.ExternalMetricValueList{
			Items: metricValues,
		}, nil
	}

	// get last 5 minutes
	endTime := time.Now()
	startTime := endTime.Add(-(query.Window))

	var apiClient *circonus.API = nil
	if c, ok := p.apiClients[query.CirconusAPIKey]; ok {
		apiClient = c
	} else {
		apiConfig := &circonus.Config{
			URL:      p.circonusAPIURL,
			TokenKey: query.CirconusAPIKey,
			TokenApp: "custom-metrics-circonus-adapter",
		}

		apiclient, err := circonus.NewAPI(apiConfig)
		if err != nil {
			return nil, err
		}
		p.apiClients[query.CirconusAPIKey] = apiclient
		apiClient = apiclient
	}

	param := map[string]interface{}{
		"period": query.Stride.Seconds(),
		"start":  startTime.Unix(),
		"end":    endTime.Unix(),
		"query":  query.CAQL,
	}

	klog.Infof("Incoming query: %s", query.CAQL)

	queryString, err := CreateURLWithQuery("/caql", param)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := apiClient.Get(queryString)
	if err != nil {
		return nil, err
	}

	klog.Infof("Response: %s", string(jsonBytes))

	var result map[string]interface{}
	_ = json.Unmarshal(jsonBytes, &result)

	if _, ok := result["_data"]; !ok {
		return nil, apierr.NewInternalError(fmt.Errorf("circonus response missing _data field"))
	}

	data := result["_data"].([]interface{})
	if len(data) == 0 {
		// This shouldn't happen with correct query to Circonus
		return nil, apierr.NewInternalError(fmt.Errorf("empty time series returned from Circonus CAQL query"))
	}

	// point is an array of [time, [value1, value2, ..., valueN]]
	// the embedded array is the output of CAQL where they can be multiple streams output.
	// this is built to deal with a single stream coming out so we only grab the first value in the embedded array.
	//
	// example return data:
	// {"_data":[[1611606180,[0]],[1611606240,[0]],[1611606300,[0]],[1611606360,[0]],[1611606420,[0]],[1611606480,[0]]],
	//  "_end":1611606540,"_period":60, ...}
	//
	// because we are using the last N minutes depending on config and data can be delayed
	//   average all N minutes of data into a single number to give to k8s.  If we return multiple
	//   it will sum them
	var finalTime float64 = 0
	var finalValue float64 = 0
	var count int = 0
	for _, p := range data {
		if p == nil {
			continue
		}
		count++
		point := p.([]interface{})
		resultEndTime := point[0].(float64)
		if resultEndTime > finalTime {
			finalTime = resultEndTime
		}
		if time.Unix(int64(resultEndTime), 0).After(endTime) {
			return nil, apierr.NewInternalError(fmt.Errorf("timeseries from Circonus has incorrect end time: %f", resultEndTime))
		}
		value := point[1].([]float64)[0]
		finalValue += value
	}
	if count == 0 {
		return nil, apierr.NewInternalError(fmt.Errorf("no datapoints found in result"))
	}
	metricValue := external_metrics.ExternalMetricValue{
		Timestamp:  metav1.NewTime(time.Unix(int64(finalTime), 0)),
		MetricName: info.Metric,
	}
	finalValue /= float64(count)
	metricValue.Value = *resource.NewMilliQuantity(int64(finalValue*1000), resource.DecimalSI)
	metricValues = append(metricValues, metricValue)

	return &external_metrics.ExternalMetricValueList{
		Items: metricValues,
	}, nil
}

// ListAllExternalMetrics returns a list of available external metrics.
// Returns the names of everything configured.
func (p *CirconusProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	l := make([]provider.ExternalMetricInfo, 0)
	for en := range p.queryMap {
		emi := provider.ExternalMetricInfo{}
		emi.Metric = en
		l = append(l, emi)
	}
	return l
}
