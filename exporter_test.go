package main

import (
	"fmt"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/jarcoal/httpmock.v1"
	"io/ioutil"
	"strings"
	"testing"
)

func TestCollectStats(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/github_repos.json")
	if err != nil {
		t.Fatal(err)
	}

	httpmock.Activate()
	defer httpmock.Deactivate()

	httpmock.RegisterNoResponder(httpmock.NewBytesResponder(200, data))

	collectStats(github.NewClient(nil))

	gathered, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}

	for _, g := range gathered {
		if strings.HasPrefix(g.GetName(), "github_") {
			for _, m := range g.GetMetric() {
				fmt.Println(g.GetName(), m.GetGauge().GetValue())
				fmt.Println(m.GetLabel())
				fmt.Println(m.String())
			}
		}
	}
}
