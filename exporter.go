package main

import (
	"context"
	"flag"
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
	"time"
)

// flags
var (
	port = flag.Int("port", 8080, "The HTTP port to listen on")
)

// metrics
type Metric struct {
	Name      string
	Extractor func(repository *github.Repository) *int

	gauge *prometheus.GaugeVec
}

func (m *Metric) Update(repository *github.Repository) {
	if value := m.Extractor(repository); value != nil {
		m.gauge.WithLabelValues(
			repository.GetOwner().GetLogin(), repository.GetName(),
		).Set(float64(*value))
	}
}

var (
	metrics []Metric
)

func addMetric(metric Metric) {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "github",
		Name:      metric.Name,
		Help:      metric.Name,
	}, []string{"owner", "repository"})

	prometheus.MustRegister(gauge)

	metric.gauge = gauge
	metrics = append(metrics, metric)
}

// prepare the metrics
func init() {
	addMetric(Metric{Name: "forks", Extractor: func(r *github.Repository) *int { return r.ForksCount }})
	addMetric(Metric{Name: "networks", Extractor: func(r *github.Repository) *int { return r.NetworkCount }})
	addMetric(Metric{Name: "open_issues", Extractor: func(r *github.Repository) *int { return r.OpenIssuesCount }})
	addMetric(Metric{Name: "stargazers", Extractor: func(r *github.Repository) *int { return r.StargazersCount }})
	addMetric(Metric{Name: "subscribers", Extractor: func(r *github.Repository) *int { return r.SubscribersCount }})
	addMetric(Metric{Name: "watchers", Extractor: func(r *github.Repository) *int { return r.WatchersCount }})
}

func collectStats(client *github.Client) {
	opts := &github.RepositoryListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, resp, err := client.Repositories.List(context.Background(), "rycus86", opts)
		if err != nil {
			panic(err)
		}

		for _, repo := range repos {
			for _, m := range metrics {
				m.Update(repo)
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}
}

func main() {
	client := github.NewClient(httpcache.NewMemoryCacheTransport().Client())

	go func() {
		for {
			select {
			case <-time.Tick(5 * time.Second):
				collectStats(client)
			}
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
