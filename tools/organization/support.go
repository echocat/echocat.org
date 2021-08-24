package main

import (
	"github.com/xanzy/go-gitlab"
	"time"
)

func pString(input string) *string {
	return &input
}

func pNonEmptyString(input string) *string {
	if len(input) == 0 {
		return nil
	}
	return &input
}

func pUint32(input uint32) *uint32 {
	return &input
}

func pTime(input time.Time) *time.Time {
	return &input
}

func pGitlabVisibilityValue(input gitlab.VisibilityValue) *gitlab.VisibilityValue {
	return &input
}
