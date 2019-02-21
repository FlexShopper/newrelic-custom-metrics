package main

import (
	"flag"
	"github.com/flexshopper/newrelic-custom-metrics/newrelic"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"

	nrProvider "github.com/flexshopper/newrelic-custom-metrics/provider"
	basecmd "github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
)

type NewrelicAdapter struct {
	basecmd.AdapterBase

	// Message is printed on successful startup
	Message string
}

type HttpGetClient struct {
	
}

func (HttpGetClient) Fetch(url string, headers map[string]string, params map[string]string) ([]byte, error) {
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return []byte{}, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}

	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}

	body, err := ioutil.ReadAll(res.Body)
	glog.Infof("Request made to %s with params %v, response was: %s", url, params, body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

func (a *NewrelicAdapter) makeProviderOrDie() provider.ExternalMetricsProvider {
	client, err := a.DynamicClient()
	if err != nil {
		glog.Fatalf("unable to construct dynamic client: %v", err)
	}

	mapper, err := a.RESTMapper()
	if err != nil {
		glog.Fatalf("unable to construct discovery REST mapper: %v", err)
	}

	newrelicApiKey := os.Getenv("NEWRELIC_API_KEY")
	if newrelicApiKey == "" {
		glog.Fatalf("NEWRELIC_API_KEY env var must be set")
	}

	minRpm := 0
	minRpmArg := os.Getenv("MIN_RPM")
	if minRpmArg != "" {
		minRpm, err = strconv.Atoi(minRpmArg)
		if err != nil {
			glog.Warningf("Could parse MIN_RPM to int, defaulting to %d", minRpm)
		}
	}

	return nrProvider.NewProvider(client, mapper, newrelic.NewApi(newrelicApiKey, minRpm, HttpGetClient{}))
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &NewrelicAdapter{}
	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the glog flags
	cmd.Flags().Parse(os.Args)

	newrelicProvider := cmd.makeProviderOrDie()
	cmd.WithExternalMetrics(newrelicProvider)

	glog.Infof(cmd.Message)

	if err := cmd.Run(wait.NeverStop); err != nil {
		glog.Fatalf("unable to run custom metrics adapter: %v", err)
	}
}
