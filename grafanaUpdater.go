package main

import (
	"fmt"
	"net/http"
	"strings"
	"errors"
)

type GrafanaUpdater struct {
	username string
	password string
	dashboardApiUrl string
	datasourceApiUrl string
}

func NewGrafanaUpdater(url string, username string, password string) *GrafanaUpdater {
	return &GrafanaUpdater{
		username: username,
		password: password,
		dashboardApiUrl: fmt.Sprintf("%s/api/dashboards/db", url),
		datasourceApiUrl: fmt.Sprintf("%s/api/datasources", url),
	}
}

func (updater *GrafanaUpdater) PushDashboard(dashboardJson string) error {
	dashboardPostBody := fmt.Sprintf("{ \"dashboard\":%s, \"overwrite\": true }", dashboardJson)
	return grafanaApiPost(updater.dashboardApiUrl, dashboardPostBody)
}

func (updater *GrafanaUpdater) PushDatasource(datasourceJson string) error {
	return grafanaApiPost(updater.datasourceApiUrl, datasourceJson)
}

func grafanaApiPost(url string, postBody string) error {
	req, err := http.NewRequest("POST", url, strings.NewReader(postBody))
	if err != nil {
		return err
	}
	req.SetBasicAuth(*grafanaUsername, *grafanaPassword)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	statusCode := resp.StatusCode
	if statusCode != 200 {
		return errors.New(fmt.Sprintf("Grafana API call failed with code %d", statusCode))
	}
	return nil
}
