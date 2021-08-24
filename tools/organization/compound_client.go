package main

import log "github.com/echocat/slf4g"

type compoundClient []client

func (instance *compoundClient) retrieveOrganization() (organization, error) {

	log.Info("Starting to retrieve the organization details...")

	var result organization
	for _, delegate := range *instance {
		if org, err := delegate.retrieveOrganization(); err != nil {
			return organization{}, err
		} else {
			result = result.merge(org)
		}
	}

	log.Info("Starting to retrieve the organization details... DONE!")

	return result, nil
}
