// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// +build go1.13

package main

import (
	"flag"
	"os"

	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"

	_ "google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/logs"
	"k8s.io/klog"

	adapter "github.com/circonus-labs/custom-metrics-circonus-adapter/pkg/adapter/provider"
	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
)

// CirconusAdapter is an adapter for Circonus CAQL
type CirconusAdapter struct {
	basecmd.AdapterBase
}

type circonusAdapterServerOptions struct {
	// the circonus provider URL
	providerAPIURLAttr string
	// the config file path
	configFile string
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

	return adapter.NewCirconusProvider(client, o.providerAPIURLAttr, o.configFile)
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
		"The Circonus API URL, defaults to https://api.circonus.com/v2")

	flags.Parse(os.Args)

	metricsProvider := cmd.makeProviderOrDie(&serverOptions)

	cmd.WithExternalMetrics(metricsProvider)

	if err := cmd.Run(wait.NeverStop); err != nil {
		klog.Fatalf("unable to run custom metrics adapter: %v", err)
	}
}
