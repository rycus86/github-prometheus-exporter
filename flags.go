package main

import (
	"flag"
	"time"
)

var (
	port      = flag.Int("port", 8080, "The HTTP port to listen on")
	interval  = flag.Duration("interval", 15*time.Minute, "Interval between checks")
	skipForks = flag.Bool("skip-forks", false, "Do not pull metrics for forked repositories")

	users multiVar
	orgs  multiVar

	usernameVar     = flag.String("username", "", "Username for authenticated API calls (optional)")
	passwordVar     = flag.String("password", "", "Password for authenticated API calls (optional)")
	credentialsFile = flag.String("credentials", "",
		"File `path` containing the authentication details in `username:password` format (optional)")
)

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

func init() {
	flag.Var(&users, "user", "Users to list repositories for (multiple values are allowed)")
	flag.Var(&orgs, "org", "Organizations to list repositories for (multiple values are allowed)")

	flag.Parse()
}
