package provider

import (
	"errors"
	"github.com/flexshopper/newrelic-custom-metrics/newrelic"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	meta1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sync"
	"time"
)

const APP_KEY = "appName"

// testingProvider is a sample implementation of provider.MetricsProvider which stores a map of fake metrics
type newrelicProvider struct {
	api newrelic.RpmProvider
	client dynamic.Interface
	mapper apimeta.RESTMapper

	valuesLock sync.RWMutex
}

func (np newrelicProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {

	appName := ""
	reqs, _ := metricSelector.Requirements()
	for _, req := range reqs {
		if req.Key() == APP_KEY {
			appName = req.Values().List()[0]
		}
	}

	if appName == "" {
		return &external_metrics.ExternalMetricValueList{}, errors.New("could not find appName selector")
	}

	rpm, err := np.api.GetApplicationRpm(appName)
	if err != nil {
		return &external_metrics.ExternalMetricValueList{}, err
	}

	return &external_metrics.ExternalMetricValueList{
		Items: []external_metrics.ExternalMetricValue{{
			MetricLabels: map[string]string{"app": namespace},
			Timestamp: meta1.Time{time.Now()},
			MetricName: "rpm",
			Value: *resource.NewQuantity(int64(rpm), resource.DecimalSI),
		}},
	}, nil
}

func (np newrelicProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	return []provider.ExternalMetricInfo{{
		Metric: "rpm",
	}}
}


// NewFakeProvider returns an instance of testingProvider, along with its restful.WebService that opens endpoints to post new fake metrics
func NewProvider(client dynamic.Interface, mapper apimeta.RESTMapper, nrApi newrelic.RpmProvider) provider.ExternalMetricsProvider {
	return &newrelicProvider{
		api: nrApi,
		client: client,
		mapper: mapper,
	}
}
