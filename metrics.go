package main

import (
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
)

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
	rateReset = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "github",
		Name:      "rate_reset",
		Help:      "API Rate Reset",
	})
)

type Metric struct {
	Name      string
	Help      string
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

func addMetric(metric Metric) {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "github",
		Name:      metric.Name,
		Help:      metric.Help,
	}, []string{"owner", "repository"})

	prometheus.MustRegister(gauge)

	metric.gauge = gauge
	metrics = append(metrics, metric)
}

func init() {
	prometheus.MustRegister(repoCount)
	prometheus.MustRegister(rateLimit)
	prometheus.MustRegister(rateRemaining)
	prometheus.MustRegister(rateReset)

	addMetric(Metric{Name: "forks_count", Help: "Number of Forks",
		Extractor: func(r *github.Repository) *int { return r.ForksCount }})
	addMetric(Metric{Name: "networks_count", Help: "Number of Networks",
		Extractor: func(r *github.Repository) *int { return r.NetworkCount }})
	addMetric(Metric{Name: "open_issues_count", Help: "Number of Open Issues",
		Extractor: func(r *github.Repository) *int { return r.OpenIssuesCount }})
	addMetric(Metric{Name: "stargazers_count", Help: "Number of Stars",
		Extractor: func(r *github.Repository) *int { return r.StargazersCount }})
	addMetric(Metric{Name: "subscribers_count", Help: "Number of Subscribers",
		Extractor: func(r *github.Repository) *int { return r.SubscribersCount }})
	addMetric(Metric{Name: "watchers_count", Help: "Number of Watchers",
		Extractor: func(r *github.Repository) *int { return r.WatchersCount }})
	addMetric(Metric{Name: "size_kilobytes", Help: "Size of the Repository in kiloBytes",
		Extractor: func(r *github.Repository) *int { return r.Size }})
}
