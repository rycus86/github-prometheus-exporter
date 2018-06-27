# Prometheus exporter for GitHub stats

A small Go application to periodically pull statistics and metrics from GitHub, and expose them in a [Prometheus](https://prometheus.io/)-compatible format.

## Usage

You can easily run the application from its [Docker image](https://hub.docker.com/r/rycus86/github-exporter/):

```shell
$ docker run --rm -it -p 8080:8080 rycus86/github-exporter <flags>
```

The command line parameters are like this:

```
Usage of /exporter:
  -credentials path
        File path containing the authentication details in `username:password` format (optional)
  -interval duration
        Interval between checks (default 15m0s)
  -org value
        Organizations to list repositories for (multiple values are allowed)
  -password string
        Password for authenticated API calls (optional)
  -port int
        The HTTP port to listen on (default 8080)
  -skip-forks
        Do not pull metrics for forked repositories
  -timeout duration
        HTTP API call timeout (default 15s)
  -user value
        Users to list repositories for (multiple values are allowed)
  -username string
        Username for authenticated API calls (optional)
```

The Docker image reference points to a multi-arch manifest, with the actual images being available for the `amd64`, `armhf` and `arm64v8` platforms.

### Selecting targets

You can specify the GitHub user or organization to list repositories from multiple times:

```shell
$ docker run --rm -it -p 8080:8080 rycus86/github-exporter \
      -user one -user two -org three -org four -org five
```

You can also add the `-skip-forks` flag to exclude any repositories the user or organization has forked, rather than creating it themselves.

### Authentication and rate limits

The application uses the [v3 GitHub API](https://developer.github.com/v3/) through the [google/go-github](https://github.com/google/go-github) library, and it allows you to either make anonymous requests to the API with lower call rate limits, or authenticated requests with higher limits. For the moment, basic authentication is supported with a provided username and password.

You can either use the `-username` and `-password` to supply the credentials, though be aware that this would make them show up on process listings with `ps` for example! A more secure way would be adding in a credentials file, perhaps with a bind-mount, or as a *secret* if you're using Docker Swarm mode.

```shell
$ docker run --rm -it -v $PWD/github.creds:/var/secret/credentials \
      rycus86/github-exporter -credentials /var/secret/credentials -user userA
```

The application also leverages the [gregjones/httpcache](https://github.com/gregjones/httpcache) library to make [conditional requests](https://developer.github.com/v3/#conditional-requests) to GitHub, which won't count against the rate limit.

## Metrics

The following metrics are exposed on the `/metrics` endpoint:

```shell
$ curl -s http://localhost:8080/metrics | grep github_
# HELP github_forks_count Number of Forks
# TYPE github_forks_count gauge
github_forks_count{owner="rycus86",repository="prometheus_flask_exporter"} 3
# HELP github_open_issues_count Number of Open Issues
# TYPE github_open_issues_count gauge
github_open_issues_count{owner="rycus86",repository="prometheus_flask_exporter"} 1
# HELP github_rate_limit API Rate Limit
# TYPE github_rate_limit gauge
github_rate_limit 60
# HELP github_rate_remaining API Rate Remaining
# TYPE github_rate_remaining gauge
github_rate_remaining 58
# HELP github_rate_reset API Rate Reset
# TYPE github_rate_reset gauge
github_rate_reset 1.530000761e+12
# HELP github_repo_count Number of Repositories
# TYPE github_repo_count gauge
github_repo_count{owner="rycus86"} 59
# HELP github_size_kilobytes Size of the Repository in kiloBytes
# TYPE github_size_kilobytes gauge
github_size_kilobytes{owner="rycus86",repository="github-prometheus-exporter"} 452
github_size_kilobytes{owner="rycus86",repository="podlike"} 2070
github_size_kilobytes{owner="rycus86",repository="prometheus_flask_exporter"} 186
# HELP github_stargazers_count Number of Stars
# TYPE github_stargazers_count gauge
github_stargazers_count{owner="rycus86",repository="podlike"} 8
github_stargazers_count{owner="rycus86",repository="prometheus_flask_exporter"} 10
# HELP github_watchers_count Number of Watchers
# TYPE github_watchers_count gauge
github_watchers_count{owner="rycus86",repository="podlike"} 8
github_watchers_count{owner="rycus86",repository="prometheus_flask_exporter"} 10
```

## Acknowledgements

The application was inspired by [infinityworks/github-exporter](https://github.com/infinityworks/github-exporter).

## License

MIT
