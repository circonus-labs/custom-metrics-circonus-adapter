package provider

import (
	"time"
)

type AdapterConfig struct {
	// Queries specifies how to name and map CAQL statements to external metrics
	Queries []Query `yaml:"queries"`
}

// Query describes a query, the api key, the name to give it and query related parameters
type Query struct {
	// CAQL specifies the statement to execute
	CAQL string `yaml:"caql"`
	// CirconusAPIKey specifies the key to use when calling the Circonus /caql endpoint
	CirconusAPIKey string `yaml:"circonus_api_key"`
	// ExternalName describes the name to give this query for purposes of referring to it
	// in the HPA config
	ExternalName string `yaml:"external_name"`
	// Window specifies the start/end times of the CAQL query fetch where end == time.Now()
	// and start == time.Now() - Window.  It allows for fetching multiple datapoints.
	// default is "5m"
	Window time.Duration `yaml:"window"`
	// Stride specifies the granularity of the data to return, defaults to "1m".
	Stride time.Duration `yaml:"stride"`
	// The function to use to combine the data in `window`, one of `average`, `min`, `max`, defaults
	// to `average`
	Aggregate string `yaml:"aggregate"`
}
