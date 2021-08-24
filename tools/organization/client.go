package main

type client interface {
	retrieveOrganization() (organization, error)
}
