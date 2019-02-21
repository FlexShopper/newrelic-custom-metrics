package provider

import (
	"errors"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
	"testing"
)

type TestRpmProvider struct {}

func (TestRpmProvider) GetApplicationRpm(appName string) (int, error) {
	if appName == "not-found" {
		return 0, errors.New("random error")
	}

	return 123, nil
}

func (TestRpmProvider) GetRPMAverageAcrossHosts(appName string) (int, error) {
	if appName == "not-found" {
		return 0, errors.New("random error")
	}

	return 123, nil
}



type TestRESTMapper struct {}

func (TestRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	panic("implement me")
}

func (TestRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	panic("implement me")
}

func (TestRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	panic("implement me")
}

func (TestRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	panic("implement me")
}

func (TestRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	panic("implement me")
}

func (TestRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	panic("implement me")
}

func (TestRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	panic("implement me")
}

type TestDynamic struct {}

func (TestDynamic) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	panic("implement me")
}

func TestGetExternalMetric (t *testing.T) {
	np := NewProvider(TestDynamic{}, TestRESTMapper{}, TestRpmProvider{})

	selector := labels.NewSelector()
	requirement, _ := labels.NewRequirement("appName", selection.Equals, []string{"fmcore"})

	selector = selector.Add(*requirement)

	valueList, err := np.GetExternalMetric("fmcore", selector, provider.ExternalMetricInfo{})

	if err != nil {
		t.Errorf("There was an error: %s", err)
	}

	if val, _ := valueList.Items[0].Value.AsInt64(); val != int64(123) {
		t.Errorf("Returned value does not match expected value")
	}
}

func TestGetExternalMetricWithApiError (t *testing.T) {
	np := NewProvider(TestDynamic{}, TestRESTMapper{}, TestRpmProvider{})

	selector := labels.NewSelector()
	requirement, _ := labels.NewRequirement("appName", selection.Equals, []string{"not-found"})

	selector = selector.Add(*requirement)

	_, err := np.GetExternalMetric("fmcore", selector, provider.ExternalMetricInfo{})

	if err.Error() != "random error" {
		t.Errorf("external metrics not emitting error from underlying api")
	}
}

func TestGetExternalMetricAppNameSelectorNotFound (t *testing.T) {
	np := NewProvider(TestDynamic{}, TestRESTMapper{}, TestRpmProvider{})

	selector := labels.NewSelector()
	requirement, _ := labels.NewRequirement("notName", selection.Equals, []string{"not-found"})

	selector = selector.Add(*requirement)

	_, err := np.GetExternalMetric("fmcore", selector, provider.ExternalMetricInfo{})

	if err.Error() != "could not find appName selector" {
		t.Errorf("error on finding appName selector")
	}
}

func TestListAllExternalMetrics (t *testing.T) {
	np := NewProvider(TestDynamic{}, TestRESTMapper{}, TestRpmProvider{})
	metricList := np.ListAllExternalMetrics()

	if len(metricList) == 0 {
		t.Errorf("no metrics returned")
	}

	if metricList[0].Metric != "rpm" {
		t.Errorf("incorrect metric returned")
	}
}