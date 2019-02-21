package newrelic

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
)

type TestApiRequest struct {}

func (TestApiRequest) Fetch(url string, headers map[string]string, params map[string]string) ([]byte, error) {

	if ok, _ := regexp.MatchString(".*applications.json$", url); ok {
		return []byte(`{"applications":[{"id":1234,"name":"marketplace"}]}`), nil
	}

	if ok, _ := regexp.MatchString(`.*applications/1234/hosts.json`, url); ok {
		return []byte(`{"application_hosts":[{"ID":245}]}`), nil
	}

	if ok, _ := regexp.MatchString(`.*applications/1234/hosts/245/metrics/data.json`, url); ok {
		return []byte(`{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"calls_per_minute":120}}]}]}}`), nil
	}

	return []byte{}, nil
}

type ApiReturn struct {
	UrlRegex string
	ReturnJson string
	ErrorReturn string
}

type TestApiRequestListAppsFails struct {
	Returns []ApiReturn
}

func (l *TestApiRequestListAppsFails) Fetch(url string, headers map[string]string, params map[string]string) ([]byte, error) {

	for _, v := range l.Returns {
		if ok, _ := regexp.MatchString(v.UrlRegex, url); ok {
			if len(v.ReturnJson) == 0 {
				return nil, errors.New(v.ErrorReturn)
			}

			return []byte(v.ReturnJson), nil
		}
	}

	return []byte{}, nil
}


func TestApi_GetRPMAverageAcrossHosts(t *testing.T) {
	nr := NewApi("123", 1, TestApiRequest{})
	rpm, _ := nr.GetRPMAverageAcrossHosts("marketplace")

	if rpm != 120 {
		t.Errorf("return RPM was not correct")
	}
}

func TestApi_GetRPMAverageAcrossHostsAppNotFound(t *testing.T) {
	nr := NewApi("123", 1 , TestApiRequest{})
	_, err := nr.GetRPMAverageAcrossHosts("not-market-place")
	if err.Error() != "could not find matching app" {
		t.Errorf("application matching not working as expected")
	}
}

func TestApi_GetRPMAverageAcrossHostsListAppFails(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{{
			UrlRegex: ".*applications.json$",
			ErrorReturn: "could not list applications",
		}},
	})
	_, err := nr.GetRPMAverageAcrossHosts("marketplace")

	if err.Error() != "could not list applications" {
		t.Errorf("failure to list applications it not bubbling up")
	}
}

func TestApi_GetRPMAverageAcrossHostsListAppFailsWithJsonError(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{not:valid:json}`,
			},
		},
	})
	_, err := nr.GetRPMAverageAcrossHosts("marketplace")
	if err == nil {
		t.Error("Invalid json is not triggering unmarshaling error")
	}
}

func TestApi_GetRPMAverageAcrossHostsCouldNotGetHosts(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ErrorReturn: "could not get hosts",
			},
		},
	})
	_, err := nr.GetRPMAverageAcrossHosts("marketplace")
	if err.Error() != "could not get hosts" {
		t.Error("there was error in getHosts")
	}
}

func TestApi_GetRPMAverageAcrossHostsCouldNotUnmarshalHosts(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: "{not:valid:json}",
			},
		},
	})
	_, err := nr.GetRPMAverageAcrossHosts("marketplace")
	if err == nil {
		t.Error("Invalid json is not triggering unmarshaling error")
	}
}

func TestApi_GetRPMAverageAcrossHostsCouldNotGetHostRpm(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: `{"application_hosts":[{"ID":245}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/245/metrics/data.json`,
				ErrorReturn: "could not get host rpm",
			},
		},
	})
	_, err := nr.GetRPMAverageAcrossHosts("marketplace")
	if err.Error() != "could not get host rpm" {
		t.Errorf("did not bubble up failure to get host rpm api call")
	}
}

func TestApi_GetRPMAverageAcrossHostsCouldNotUnmarshalJson(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: `{"application_hosts":[{"ID":245}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/245/metrics/data.json`,
				ReturnJson: "{not:valid:json}",
			},
		},
	})
	_, err := nr.GetRPMAverageAcrossHosts("marketplace")
	if err == nil {
		t.Error("Invalid json is not triggering unmarshaling error")
	}
}

func TestApi_GetRPMAverageAcrossHostUnmarshalsFloatsToInts(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: `{"application_hosts":[{"ID":245}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/245/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"calls_per_minute":12.5}}]}]}}`,
			},
		},
	})

	rpm, _ := nr.GetRPMAverageAcrossHosts("marketplace")
	if rpm != 12 {
		t.Errorf("Expected rpm of 12, got %d", rpm)
	}
}

func TestApi_GetRPMAverageAcrossHostsCouldNotUnmarshalBadValueToFloat(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: `{"application_hosts":[{"ID":245}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/245/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"calls_per_minute":"not.a.float"}}]}]}}`,
			},
		},
	})

	_, err := nr.GetRPMAverageAcrossHosts("marketplace")
	if err == nil {
		t.Errorf("Did not detect float conversion error")
	}
}

func TestApi_GetRPMAverageAcrossHostsCouldNotUnmarshalBadValueToInt(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: `{"application_hosts":[{"ID":245}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/245/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"calls_per_minute":"not an int"}}]}]}}`,
			},
		},
	})

	rpm, _ := nr.GetRPMAverageAcrossHosts("marketplace")
	if rpm != 0 {
		// @todo find a way to force a bad int
		t.Errorf("something odd happened with bad int conversion")
	}
}

func TestApi_GetRPMAverageAcrossHostsUnmarshalsIntsToInts(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: `{"application_hosts":[{"ID":245}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/245/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"calls_per_minute":250}}]}]}}`,
			},
		},
	})

	rpm, _ := nr.GetRPMAverageAcrossHosts("marketplace")
	if rpm != 250 {
		t.Errorf("Expected rpm of 250, got %d", rpm)
	}
}

func TestApi_GetRPMAverageAcrossHostsIgnoresHostRpmBelowMinRpm(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts.json`,
				ReturnJson: `{"application_hosts":[{"ID":245}, {"ID":246}]}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/245/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"calls_per_minute":250}}]}]}}`,
			},
			{
				UrlRegex: `.*applications/1234/hosts/246/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"calls_per_minute":0}}]}]}}`,
			},
		},
	})

	rpm, _ := nr.GetRPMAverageAcrossHosts("marketplace")
	if rpm != 250 {
		t.Errorf("Expected rpm of 250, got %d", rpm)
	}
}

func TestApi_GetApplicationRpm(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"requests_per_minute":250}}]}]}}`,
			},
		},
	})

	rpm, err := nr.GetApplicationRpm("marketplace")
	fmt.Printf("%v", err)
	if rpm != 250 {
		t.Errorf("Expected rpm of 250, got %d", rpm)
	}
}

func TestApi_GetApplicationRpmCouldNotFindApplication(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"not-marketplace"}]}`,
			},
		},
	})

	_, err := nr.GetApplicationRpm("marketplace")
	if err.Error() != "could not find matching app" {
		t.Error("error for not finding application not coming through")
	}
}

func TestApi_GetApplicationRpmApiRequestError(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/metrics/data.json`,
				ErrorReturn: `api request failed`,
			},
		},
	})

	_, err := nr.GetApplicationRpm("marketplace")
	if err.Error() != "api request failed" {
		t.Error("failing to stop on api error")
	}
}

func TestApi_GetApplicationRpmInvalidJson(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/metrics/data.json`,
				ReturnJson: `{not:valid:json}`,
			},
		},
	})

	_, err := nr.GetApplicationRpm("marketplace")
	if err == nil {
		t.Error("invalid json is not returning an error")
	}
}

func TestApi_GetApplicationUnmarshalsFloatsToInts(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"requests_per_minute":24.5}}]}]}}`,
			},
		},
	})

	rpm, _ := nr.GetApplicationRpm("marketplace")
	if rpm != 24 {
		t.Errorf("Expected rpm of 24, got %d", rpm)
	}
}

func TestApi_GetApplicationsCouldNotUnmarshalBadValueToFloat(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"requests_per_minute":"not.a.float"}}]}]}}`,
			},
		},
	})

	_, err := nr.GetApplicationRpm("marketplace")
	if err == nil {
		t.Errorf("Did not detect float conversion error")
	}
}

func TestApi_GetApplicationsCouldNotUnmarshalBadValueToInt(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"requests_per_minute":"not an int"}}]}]}}`,
			},
		},
	})

	rpm, _ := nr.GetApplicationRpm("marketplace")
	if rpm != 0 {
		// @todo find a way to force a bad int
		t.Errorf("something odd happened with bad int conversion")
	}
}

func TestApi_GetApplicationsUnmarshalsIntsToInts(t *testing.T) {
	nr := NewApi("123", 1, &TestApiRequestListAppsFails{
		Returns: []ApiReturn{
			{
				UrlRegex: ".*applications.json$",
				ReturnJson: `{"applications":[{"id":1234,"name":"marketplace"}]}`,
			},
			{
				UrlRegex: `.*applications/1234/metrics/data.json`,
				ReturnJson: `{"metric_data":{"from":"foo","to":"foo","metrics_not_found":[],"metrics_found":["HttpDispatcher"],"metrics":[{"name":"HttpDispatcher","timeslices":[{"from":"foo","to":"foo","values":{"requests_per_minute":250}}]}]}}`,
			},
		},
	})

	rpm, _ := nr.GetApplicationRpm("marketplace")
	if rpm != 250 {
		t.Errorf("Expected rpm of 250, got %d", rpm)
	}
}