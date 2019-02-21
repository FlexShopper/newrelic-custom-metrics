package newrelic

import (
	"encoding/json"
	"errors"
	"github.com/golang/glog"
	"strconv"
	"strings"
)

type applicationEntry struct {
	ID int `json:"id"`
	Name string `json:"name"`
}

type applicationList struct {
	Applications []applicationEntry `json:"applications"`
}

type timeSlice struct {
	From string `json:"from"`
	To string `json:"to"`
	Values map[string]json.Number `json:"values"`
}

type metric struct {
	Name string `json:"name"`
	TimeSlices []timeSlice `json:"timeslices"`
}

type metricList []metric

type metricsData struct {
	From string `json:"from"`
	To string `json:"to"`
	MetricsNotFound []string `json:"metrics_not_found"`
	MetricsFound []string `json:"metrics_found"`
	Metrics metricList `json:"metrics"`
}

type metricsDataResponse struct {
	MetricsData metricsData `json:"metric_data"`
}

type applicationHost struct {
	ID int `json:"id"`
}

type applicationHostResponse struct {
	Hosts []applicationHost `json:"application_hosts"`
}

type GetApiRequest interface {
	Fetch(url string, headers map[string]string, params map[string]string) ([]byte, error)
}

type Api struct {
	baseUri string
	minRpmForConsideration int
	apiKey string
	httpClient GetApiRequest
}

type RpmProvider interface {
	GetApplicationRpm(appName string) (int, error)
}

func NewApi(apiKey string, minRpmForConsideration int, client GetApiRequest) *Api {
	return &Api{
		baseUri: "https://api.newrelic.com/v2/",
		apiKey: apiKey,
		minRpmForConsideration: minRpmForConsideration,
		httpClient: client,
	}
}

func (nr *Api) apiRequest(uri string, queryParams map[string]string) ([]byte, error) {
	headers := map[string]string{
		"x-api-key": nr.apiKey,
		"content-type": "application/json",
	}

	return nr.httpClient.Fetch(uri, headers, queryParams)
}

func (nr *Api) listApps() (applicationList, error) {
	body, err := nr.apiRequest(nr.baseUri + "applications.json", map[string]string{})

	if err != nil {
		return applicationList{}, err
	}

	appList := applicationList{}
	err = json.Unmarshal(body, &appList)
	if err != nil {
		return applicationList{}, err
	}

	return appList, nil
}

func (nr *Api) getHostsForApp(appId int) (applicationHostResponse, error) {
	uri := nr.baseUri + "applications/"+ strconv.Itoa(appId) +"/hosts.json"
	body, err := nr.apiRequest(uri, map[string]string{})

	if err != nil {
		return applicationHostResponse{}, err
	}

	appHosts := applicationHostResponse{}
	err = json.Unmarshal(body, &appHosts)
	if err != nil {
		return applicationHostResponse{}, err
	}

	return appHosts, nil
}

func (nr *Api) getApplicationId(appName string) (int, error) {
	apps, err := nr.listApps()
	if err != nil {
		return 0, err
	}

	appId := 0
	for _, app := range apps.Applications {
		if app.Name == appName {
			appId = app.ID
			break
		}
	}

	if appId == 0 {
		return 0, errors.New("could not find matching app")
	}

	return appId, nil
}

func (nr *Api) parseInt(value json.Number) (int, error) {
	intVal := 0
	if strings.Contains(value.String(), ".") {
		floatRpm, err := value.Float64()
		if err != nil {
			return 0, err
		}

		intVal = int(floatRpm)
	} else {
		rpmLarge, err := value.Int64()
		if err != nil {
			return 0, nil
		}

		intVal = int(rpmLarge)
	}

	return intVal, nil
}

func (nr *Api) GetApplicationRpm(appName string) (int, error) {
	appId, err := nr.getApplicationId(appName)
	if err != nil {
		return 0, err
	}

	uri := nr.baseUri + "applications/"+ strconv.Itoa(appId) +"/metrics/data.json"
	params := map[string]string{
		"names[]": "HttpDispatcher",
		"values[]": "requests_per_minute",
		"summarize": "true",
	}

	body, err := nr.apiRequest(uri, params)

	if err != nil {
		return 0, err
	}

	appMetrics := metricsDataResponse{}
	err = json.Unmarshal(body, &appMetrics)
	if err != nil {
		return 0, err
	}

	cpm := appMetrics.MetricsData.Metrics[0].TimeSlices[0].Values["requests_per_minute"]
	return nr.parseInt(cpm)
}

func (nr *Api) getHostRpm(hostId int, appId int) (int, error) {
	uri := nr.baseUri + "applications/"+ strconv.Itoa(appId) +"/hosts/"+ strconv.Itoa(hostId) +"/metrics/data.json"
	params := map[string]string{
		"names[]": "HttpDispatcher",
		"values[]": "calls_per_minute",
		"summarize": "true",
	}

	body, err := nr.apiRequest(uri, params)

	if err != nil {
		return 0, err
	}

	hostMetrics := metricsDataResponse{}
	err = json.Unmarshal(body, &hostMetrics)
	if err != nil {
		return 0, err
	}

	cpm := hostMetrics.MetricsData.Metrics[0].TimeSlices[0].Values["calls_per_minute"]
	return nr.parseInt(cpm)
}

func (nr *Api) GetRPMAverageAcrossHosts(appName string) (int, error) {
	appId, err := nr.getApplicationId(appName)
	if err != nil {
		return 0, err
	}

	hosts, err := nr.getHostsForApp(appId)
	if err != nil {
		return 0, err
	}

	totalRpm := 0
	consideredHosts := 0
	for _, host := range hosts.Hosts {
		hostRpm, err := nr.getHostRpm(host.ID, appId)
		if err != nil {
			return 0, err
		}

		if hostRpm >= nr.minRpmForConsideration {
			consideredHosts++
			totalRpm += hostRpm
		}
	}

	if consideredHosts == 0 {
		glog.Warningf("No hosts were found to be above the minimum RPM of %d", nr.minRpmForConsideration)
		return 0, nil
	}

	return int(totalRpm / consideredHosts), nil
}