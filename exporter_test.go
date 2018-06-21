package main

import (
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"gopkg.in/jarcoal/httpmock.v1"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCollectStatsForUser(t *testing.T) {
	for _, m := range metrics {
		m.gauge.Reset()
	}

	users = multiVar([]string{"rycus86"})
	orgs = multiVar{}

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
				if name == "github_watchers_count" && value != 7.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				if name == "github_forks_count" && value != 2.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				tests["docker-prometheus"] = 1
			}

			if labelMatches(labels, "repository", "TweetWear") {
				if name == "github_open_issues_count" && value != 13.0 {
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

func TestCollectStatsForOrg(t *testing.T) {
	for _, m := range metrics {
		m.gauge.Reset()
	}

	users = multiVar{}
	orgs = multiVar([]string{"docker"})

	data, err := ioutil.ReadFile("testdata/org_repos.json")
	if err != nil {
		t.Fatal(err)
	}

	httpmock.Activate()
	defer httpmock.Deactivate()

	httpmock.RegisterResponder(
		"GET", "https://api.github.com/orgs/docker/repos",
		httpmock.NewBytesResponder(200, data))

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

			if labelMatches(labels, "repository", "docker-py") {
				if name == "github_watchers_count" && value != 3081.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				if name == "github_forks_count" && value != 1103.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				tests["docker-py"] = 1
			}

			if labelMatches(labels, "repository", "compose") {
				if name == "github_open_issues_count" && value != 585.0 {
					t.Error("Unexpected value:", name, m.String())
				}

				tests["compose"] = 1
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

func TestUnauthenticatedClient(t *testing.T) {
	client := getApiClient()

	if _, ok := client.Transport.(*httpcache.Transport); !ok {
		t.Errorf("Unexpected API client transport: %T\n", client.Transport)
	}
}

func TestAuthenticatedClientByUsernameAndPassword(t *testing.T) {
	*usernameVar = "example"
	*passwordVar = "p4$$w0rd"

	client := getApiClient()

	if tp, ok := client.Transport.(*github.BasicAuthTransport); !ok {
		t.Errorf("Unexpected API client transport: %T\n", client.Transport)
	} else if tp.Username != "example" || tp.Password != "p4$$w0rd" {
		t.Errorf("Invalid username/password found: %s:%s", tp.Username, tp.Password)
	}
}

func TestAuthenticatedClientByCredentials(t *testing.T) {
	if tf, err := ioutil.TempFile("", "gh-exporter-creds"); err != nil {
		t.Fatal("Failed to create a temporary file:", err)
	} else {
		defer os.Remove(tf.Name())

		tf.WriteString("from:cr3d3nt14l$")
		tf.Close()

		*credentialsFile = tf.Name()
	}

	client := getApiClient()

	if tp, ok := client.Transport.(*github.BasicAuthTransport); !ok {
		t.Errorf("Unexpected API client transport: %T\n", client.Transport)
	} else if tp.Username != "from" || tp.Password != "cr3d3nt14l$" {
		t.Errorf("Invalid username/password found: %s:%s", tp.Username, tp.Password)
	}
}
