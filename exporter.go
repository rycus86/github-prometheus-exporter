package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// multi-valued flag type
type multiVar []string

func (mv *multiVar) Set(value string) error {
	*mv = append(*mv, value)
	return nil
}

func (mv *multiVar) String() string {
	all := ""
	for _, item := range *mv {
		if all != "" {
			all += ", "
		}

		all += item
	}

	return "[" + all + "]"
}

// flags
var (
	port     = flag.Int("port", 8080, "The HTTP port to listen on (default: 8080)")
	interval = flag.Duration("interval", 1*time.Hour, "Interval between checks (default: 1h)")
	owners   multiVar
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

	repoCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "github",
		Name:      "repo_count",
		Help:      "Number of Repositories",
	}, []string{"owner"})

	rateLimit = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "github",
		Name:      "rate_limit",
		Help:      "API Rate Limit",
	})
	rateRemaining = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "github",
		Name:      "rate_remaining",
		Help:      "API Rate Remaining",
	})
)

func addMetric(metric Metric) {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "github",
		Name:      metric.Name + "_count",
		Help:      "Number of " + strings.Title(strings.Replace(metric.Name, "_", " ", -1)),
	}, []string{"owner", "repository"})

	prometheus.MustRegister(gauge)

	metric.gauge = gauge
	metrics = append(metrics, metric)
}

func init() {
	// multi-value flags
	flag.Var(&owners, "owner", "Repository owners to look for (multiple values are allowed)")
	flag.Parse()

	// prepare the metrics
	prometheus.MustRegister(repoCount)
	prometheus.MustRegister(rateLimit)
	prometheus.MustRegister(rateRemaining)

	addMetric(Metric{Name: "forks", Extractor: func(r *github.Repository) *int { return r.ForksCount }})
	addMetric(Metric{Name: "networks", Extractor: func(r *github.Repository) *int { return r.NetworkCount }})
	addMetric(Metric{Name: "open_issues", Extractor: func(r *github.Repository) *int { return r.OpenIssuesCount }})
	addMetric(Metric{Name: "stargazers", Extractor: func(r *github.Repository) *int { return r.StargazersCount }})
	addMetric(Metric{Name: "subscribers", Extractor: func(r *github.Repository) *int { return r.SubscribersCount }})
	addMetric(Metric{Name: "watchers", Extractor: func(r *github.Repository) *int { return r.WatchersCount }})
}

func collectStats(client *github.Client) {
	for _, owner := range owners {
		log.Println("Collecting metrics for", owner)

		collectStatsFor(owner, client)
	}
}

func collectStatsFor(owner string, client *github.Client) {
	totalCount := 0

	opts := &github.RepositoryListOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, resp, err := client.Repositories.List(context.Background(), owner, opts)
		if err != nil {
			fmt.Println("Failed to fetch page", opts.Page, "of the repos for", owner, ":", err)
			return
		}

		rateLimit.Set(float64(resp.Rate.Limit))
		rateRemaining.Set(float64(resp.Rate.Remaining))

		totalCount += len(repos)

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

	repoCount.WithLabelValues(owner).Set(float64(totalCount))
}

func main() {
	if len(owners) == 0 {
		log.Fatal("No repository owners were defined")
	}

	client := github.NewClient(httpcache.NewMemoryCacheTransport().Client())

	go func() {
		firstRun := time.After(0 * time.Second)

		for {
			select {
			case <-firstRun:
				collectStats(client)

			case <-time.Tick(*interval):
				collectStats(client)
			}
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
