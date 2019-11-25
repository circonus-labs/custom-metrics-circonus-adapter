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

package main

import (
	"flag"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"os"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/logs"
	"k8s.io/klog"

	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	adapter "github.com/rileyberton/custom-metrics-circonus-adapter/pkg/adapter/provider"
)

// StackdriverAdapter is an adapter for Stackdriver
type CirconusAdapter struct {
	basecmd.AdapterBase
}

type circonusAdapterServerOptions struct {
	// the circonus provider URL
	providerAPIURLAttr string
}

func (a *CirconusAdapter) makeProviderOrDie(o *circonusAdapterServerOptions) provider.MetricsProvider {
	config, err := a.ClientConfig()
	if err != nil {
		klog.Fatalf("unable to construct client config: %v", err)
	}

	client, err := coreclient.NewForConfig(config)
	if err != nil {
		klog.Fatalf("unable to construct client: %v", err)
	}

	return adapter.NewCirconusProvider(client, o.providerAPIURLAttr)
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &CirconusAdapter{
		basecmd.AdapterBase{
			Name: "custom-metrics-circonus-adapter",
		},
	}
	flags := cmd.Flags()

	flags.AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags

	serverOptions := circonusAdapterServerOptions{
		providerAPIURLAttr: "https://api.circonus.com/v2",
	}

	flags.StringVar(&serverOptions.providerAPIURLAttr, "circonus-api-url", serverOptions.providerAPIURLAttr,
		"whether to use new Stackdriver resource model")

	flags.Parse(os.Args)

	metricsProvider := cmd.makeProviderOrDie(&serverOptions)

	cmd.WithExternalMetrics(metricsProvider)

	if err := cmd.Run(wait.NeverStop); err != nil {
		klog.Fatalf("unable to run custom metrics adapter: %v", err)
	}
}
