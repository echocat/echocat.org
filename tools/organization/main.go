package main

import (
	"flag"
	_ "github.com/echocat/slf4g"
	log "github.com/echocat/slf4g"
	_ "github.com/echocat/slf4g/native"
	"os"
)

var (
	output = flag.String("output", "organization.json", "JSON file where to store the retrieved organization inside.")
)

func main() {
	flag.Parse()
	var err error

	assetClient := newAssetClient()
	if err := assetClient.cleanTarget(); err != nil {
		log.WithError(err).
			Fatal("Cannot clean target.")
		os.Exit(1)
	}

	client := &compoundClient{
		newGithubClient(assetClient),
		newGitlabClient(assetClient),
	}
	org, err := client.retrieveOrganization()
	if err != nil {
		log.WithError(err).
			Fatal("Cannot start database.")
		os.Exit(1)
	}

	if err := org.save(*output); err != nil {
		log.WithError(err).
			Fatal("Cannot start database.")
		os.Exit(1)
	}

}
