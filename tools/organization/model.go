package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type organization struct {
	Members    members    `json:"members"`
	Projects   projects   `json:"projects"`
	Statistics statistics `json:"statistics"`
}

func (instance organization) merge(with ...organization) (result organization) {
	result = instance

	membersAsMap := map[string]member{}
	for _, member := range instance.Members {
		membersAsMap[member.Name] = member
	}

	for _, in := range with {
		result.Projects = append(result.Projects, in.Projects...)
		for _, member := range in.Members {
			if existing, ok := membersAsMap[member.Name]; ok {
				membersAsMap[member.Name] = existing.merge(member)
			} else {
				membersAsMap[member.Name] = member
			}
		}
	}

	result.Members = []member{}
	for _, member := range membersAsMap {
		result.Members = append(result.Members, member)
	}

	result.align()
	return
}

func (instance organization) membersAsMap() map[string]member {
	result := map[string]member{}
	for _, member := range instance.Members {
		result[member.Name] = member
	}
	return result
}

func (instance *organization) align() {
	for _, project := range instance.Projects {
		instance.Statistics.NumberOfProjects++
		if project.NumberOfOpenIssues != nil {
			instance.Statistics.NumberOfOpenIssues += *project.NumberOfOpenIssues
		}
		if project.NumberOfStars != nil {
			instance.Statistics.NumberOfStars += *project.NumberOfStars
		}
		if project.NumberOfWatchers != nil {
			instance.Statistics.NumberOfWatchers += *project.NumberOfWatchers
		}
		if project.NumberOfForks != nil {
			instance.Statistics.NumberOfForks += *project.NumberOfForks
		}
	}
	instance.Statistics.NumberOfMembers = uint32(len(instance.Members))
	sort.Sort(instance.Members)
	sort.Sort(instance.Projects)
}

func (instance organization) save(to string) (err error) {
	if err := os.MkdirAll(filepath.Dir(to), 755); err != nil {
		return fmt.Errorf("cannot create directory of '%s' to store the content inside: %v", to, err)
	}
	if f, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return fmt.Errorf("cannot open '%s' to store the content inside: %v", to, err)
	} else {
		defer func() {
			if cErr := f.Close(); cErr != nil && err == nil {
				err = cErr
			}
		}()

		encoder := json.NewEncoder(f)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(instance); err != nil {
			return fmt.Errorf("cannot write '%s' to store the content inside: %v", to, err)
		}

		return nil
	}

}

type project struct {
	Type               string     `json:"type"`
	Origin             string     `json:"origin"`
	Fullname           string     `json:"fullname"`
	Name               string     `json:"name"`
	Description        *string    `json:"description"`
	DefaultBranch      *string    `json:"defaultBranch"`
	Language           *string    `json:"language"`
	HomepageUrl        *string    `json:"homepageUrl"`
	ImageUrl           *string    `json:"imageUrl"`
	ProfileUrl         string     `json:"profileUrl"`
	HttpCloneUrl       *string    `json:"httpCloneUrl"`
	SshCloneUrl        *string    `json:"sshCloneUrl"`
	IssuesUrl          *string    `json:"issuesUrl"`
	WikiUrl            *string    `json:"wikiUrl"`
	ForksUrl           *string    `json:"forksUrl"`
	PullRequestsUrl    *string    `json:"pullRequestsUrl"`
	CreateForkUrl      *string    `json:"createForkUrl"`
	StarsUrl           *string    `json:"starsUrl"`
	WatchersUrl        *string    `json:"watchersUrl"`
	NumberOfForks      *uint32    `json:"numberOfForks"`
	NumberOfOpenIssues *uint32    `json:"numberOfOpenIssues"`
	NumberOfStars      *uint32    `json:"numberOfStars"`
	NumberOfWatchers   *uint32    `json:"numberOfWatchers"`
	CreatedAt          *time.Time `json:"createdAt"`
	UpdatedAt          *time.Time `json:"updatedAt"`
}

type projects []project

func (a projects) Len() int      { return len(a) }
func (a projects) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a projects) Less(i, j int) bool {
	if a[i].UpdatedAt == nil && a[j].UpdatedAt == nil {
		return false
	}
	if a[i].UpdatedAt == nil {
		return false
	}
	if a[j].UpdatedAt == nil {
		return true
	}
	return a[i].UpdatedAt.After(*a[j].UpdatedAt)
}

type member struct {
	Type        string     `json:"type"`
	Fullname    string     `json:"fullname"`
	Name        string     `json:"name"`
	Email       *string    `json:"email"`
	ImageUrl    string     `json:"imageUrl"`
	ProfileUrl  string     `json:"profileUrl"`
	Bio         *string    `json:"bio"`
	Location    *string    `json:"location"`
	Company     *string    `json:"company"`
	HomepageUrl *string    `json:"homepageUrl"`
	SkypeId     *string    `json:"skypeId"`
	LinkedinId  *string    `json:"linkedinId"`
	TwitterId   *string    `json:"twitterId"`
	CreatedAt   *time.Time `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`
}

func (a members) Len() int           { return len(a) }
func (a members) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a members) Less(i, j int) bool { return a[i].Fullname < a[j].Fullname }

type members []member

func (instance member) merge(with ...member) member {
	result := instance

	for _, in := range with {
		if result.Email == nil && in.Email != nil {
			result.Email = pString(*in.Email)
		}
		if result.Bio == nil && in.Bio != nil {
			result.Bio = pString(*in.Bio)
		}
		if result.Location == nil && in.Location != nil {
			result.Location = pString(*in.Location)
		}
		if result.Company == nil && in.Company != nil {
			result.Company = pString(*in.Company)
		}
		if result.HomepageUrl == nil && in.HomepageUrl != nil {
			result.HomepageUrl = pString(*in.HomepageUrl)
		}
		if result.SkypeId == nil && in.SkypeId != nil {
			result.SkypeId = pString(*in.SkypeId)
		}
		if result.LinkedinId == nil && in.LinkedinId != nil {
			result.LinkedinId = pString(*in.LinkedinId)
		}
		if result.TwitterId == nil && in.TwitterId != nil {
			result.TwitterId = pString(*in.TwitterId)
		}
	}

	return result
}

type statistics struct {
	NumberOfMembers    uint32 `json:"numberOfMembers"`
	NumberOfProjects   uint32 `json:"numberOfRepositories"`
	NumberOfStars      uint32 `json:"numberOfStars"`
	NumberOfOpenIssues uint32 `json:"numberOfOpenIssues"`
	NumberOfWatchers   uint32 `json:"numberOfWatchers"`
	NumberOfForks      uint32 `json:"numberOfForks"`
}
