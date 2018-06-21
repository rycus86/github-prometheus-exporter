package main

import (
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"gopkg.in/jarcoal/httpmock.v1"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestCollectStats(t *testing.T) {
	owners = multiVar([]string{"rycus86"})

	data, err := ioutil.ReadFile("testdata/repos_p1.json")
	if err != nil {
		t.Fatal(err)
	}
	pageOneResponse := httpmock.NewBytesResponse(200, data)

	data, err = ioutil.ReadFile("testdata/repos_p2.json")
	if err != nil {
		t.Fatal(err)
	}
	pageTwoResponse := httpmock.NewBytesResponse(200, data)

	httpmock.Activate()
	defer httpmock.Deactivate()

	pageOneResponse.Header.Set("Link", "<https://api.github.com/users/rycus86/repos?page=2>; rel=\"next\"")

	httpmock.RegisterResponder(
		"GET", "https://api.github.com/users/rycus86/repos",
		func(req *http.Request) (*http.Response, error) {
			if req.URL.Query().Get("page") == "2" {
				return pageTwoResponse, nil
			} else {
				return pageOneResponse, nil
			}
		})

	collectStats(github.NewClient(nil))

	gathered, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]int{}

	for _, g := range gathered {
		name := g.GetName()

		if !strings.HasPrefix(name, "github_") {
			continue
		}

		for _, m := range g.GetMetric() {
			labels := m.GetLabel()
			value := m.GetGauge().GetValue()

			if labelMatches(labels, "repository", "docker-prometheus") {
				if name == "github_watchers" && value != 7.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				if name == "github_forks" && value != 2.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				tests["docker-prometheus"] = 1
			}

			if labelMatches(labels, "repository", "TweetWear") {
				if name == "github_open_issues" && value != 13.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				tests["TweetWear"] = 1
			}
		}
	}

	if len(tests) != 2 {
		t.Error("Only checked", len(tests), "metrics, but expected 2")
	}
}

func labelMatches(labels []*dto.LabelPair, name, value string) bool {
	for _, label := range labels {
		if label.GetName() == name {
			return label.GetValue() == value
		}
	}

	return false
}
