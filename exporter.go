package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(users) == 0 && len(orgs) == 0 {
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println()

		log.Fatal("No users or organizations were defined")
	}

	client := github.NewClient(getApiClient())

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

func getApiClient() *http.Client {
	var (
		username string
		password string
	)

	if *credentialsFile != "" {
		if contents, err := ioutil.ReadFile(*credentialsFile); err != nil {
			log.Fatalln("Failed to read the credentials file:", err)
		} else {
			parts := strings.SplitN(string(contents), ":", 2)
			username = strings.TrimSpace(parts[0])
			password = strings.TrimSpace(parts[1])
		}
	} else if *usernameVar != "" && *passwordVar != "" {
		username = *usernameVar
		password = *passwordVar
	}

	if username != "" && password != "" {
		return &http.Client{
			Transport: &github.BasicAuthTransport{
				Username:  username,
				Password:  password,
				Transport: httpcache.NewMemoryCacheTransport(),
			},
			Timeout: *timeout,
		}
	} else {
		return &http.Client{
			Transport: httpcache.NewMemoryCacheTransport(),
			Timeout:   *timeout,
		}
	}
}

func collectStats(client *github.Client) {
	for _, user := range users {
		log.Println("Collecting metrics for", user)

		collectStatsFor(user,
			func(opts github.ListOptions) ([]*github.Repository, *github.Response, error) {
				return client.Repositories.List(
					context.Background(), user, &github.RepositoryListOptions{ListOptions: opts})
			})
	}

	for _, org := range orgs {
		log.Println("Collecting metrics for", org)

		collectStatsFor(org,
			func(opts github.ListOptions) ([]*github.Repository, *github.Response, error) {
				return client.Repositories.ListByOrg(
					context.Background(), org, &github.RepositoryListByOrgOptions{ListOptions: opts})
			})
	}
}

func collectStatsFor(owner string, listFunc func(github.ListOptions) ([]*github.Repository, *github.Response, error)) {
	totalCount := 0

	opts := github.ListOptions{PerPage: 100}

	for {
		repos, resp, err := listFunc(opts) // client.Repositories.List(context.Background(), user, opts)
		if err != nil {
			log.Println("Failed to fetch page ", opts.Page, " of the repos for ", owner, ": ", err)
			return
		}

		// update rate limit related metrics
		rateLimit.Set(float64(resp.Rate.Limit))
		rateRemaining.Set(float64(resp.Rate.Remaining))
		rateReset.Set(float64(resp.Reset.UnixNano() / time.Millisecond.Nanoseconds()))

		// update the repo metrics
		for _, repo := range repos {
			if *skipForks && repo.GetFork() {
				continue
			}

			// keep track of the total number of repos
			totalCount += 1

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
