package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/xanzy/go-gitlab"
)

var (
	gitlabEntriesPerPage         = flag.Int("gitlab-entriesPerPage", 50, "")
	gitlabMaximumNumberOfEntries = flag.Int("gitlab-maximumNumberOfEntries", -1, "")
	gitlabAccessToken            = flag.String("gitlabAccessToken", "", "Gitlab accessToken to access the API.")
)

type gitlabClient struct {
	accessToken string
	assetClient *assetClient
}

type gitlabClientRetrieveTask struct {
	*gitlabClient

	client *gitlab.Client
	ctx    context.Context
}

func newGitlabClient(assetClient *assetClient) *gitlabClient {
	return &gitlabClient{
		assetClient: assetClient,
		accessToken: *gitlabAccessToken,
	}
}

func (instance *gitlabClient) retrieveOrganization() (organization, error) {
	ctx := context.Background()

	c, err := instance.newClient(ctx)
	if err != nil {
		return organization{}, fmt.Errorf("cannot create GitLab cient: %w", err)
	}

	task := gitlabClientRetrieveTask{
		gitlabClient: instance,
		client:       c,
		ctx:          ctx,
	}

	return task.execute()
}

func (instance *gitlabClientRetrieveTask) execute() (organization, error) {
	if projects, err := instance.retrieveProjects(); err != nil {
		return organization{}, err
	} else if members, err := instance.retrieveMembers(); err != nil {
		return organization{}, err
	} else {
		result := organization{
			Projects: projects,
			Members:  members,
		}
		result.align()
		return result, nil
	}
}

func (instance *gitlabClientRetrieveTask) retrieveMembers() ([]member, error) {
	var result []member
	opt := &gitlab.ListGroupMembersOptions{
		ListOptions: gitlab.ListOptions{PerPage: *gitlabEntriesPerPage},
	}
	for i := 1; *gitlabMaximumNumberOfEntries < 0 || i < *gitlabMaximumNumberOfEntries; {
		groupMembers, resp, err := instance.client.Groups.ListGroupMembers(3460920, opt)
		if err != nil {
			return result, fmt.Errorf("cannot search for group members: %v", err)
		}
		for _, groupMember := range groupMembers {
			if *gitlabMaximumNumberOfEntries > 0 && i > *gitlabMaximumNumberOfEntries {
				break
			}
			if project, err := instance.groupMemberToMember(*groupMember); err != nil {
				return nil, fmt.Errorf("cannot get details of group member '%s': %w", groupMember.Username, err)
			} else {
				result = append(result, project)
			}
			i++
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return result, nil
}

func (instance *gitlabClientRetrieveTask) detailsOfGroupMember(input gitlab.GroupMember) (gitlab.User, error) {
	if result, _, err := instance.client.Users.GetUser(input.ID, gitlab.GetUsersOptions{}); err != nil {
		return gitlab.User{}, fmt.Errorf("cannot get details of GitLab user %s(%d): %v", input.Username, input.ID, err)
	} else {
		return *result, nil
	}
}

func (instance *gitlabClientRetrieveTask) groupMemberToMember(repo gitlab.GroupMember) (member, error) {
	if detailed, err := instance.detailsOfGroupMember(repo); err != nil {
		return member{}, err
	} else {
		name := detailed.Username
		fullname := detailed.Name
		if len(fullname) == 0 {
			fullname = name
		}
		profile := "https://gitlab.com/" + detailed.Username
		homepage := detailed.WebsiteURL
		if len(homepage) == 0 {
			homepage = profile
		}
		imageAsset := ""
		if detailed.AvatarURL != "" {
			r, err := instance.assetClient.retrieve(detailed.AvatarURL)
			if err != nil {
				return member{}, err
			}
			imageAsset = r
		}

		return member{
			Type:        "user:gitlab",
			Fullname:    fullname,
			Name:        name,
			Email:       pNonEmptyString(detailed.Email),
			ImageAsset:  imageAsset,
			ProfileUrl:  profile,
			Bio:         pNonEmptyString(detailed.Bio),
			Location:    pNonEmptyString(detailed.Location),
			Company:     pNonEmptyString(detailed.Organization),
			TwitterId:   pNonEmptyString(detailed.Twitter),
			SkypeId:     pNonEmptyString(detailed.Skype),
			LinkedinId:  pNonEmptyString(detailed.Linkedin),
			HomepageUrl: &homepage,
			CreatedAt:   detailed.CreatedAt,
		}, nil
	}
}

func (instance *gitlabClientRetrieveTask) retrieveProjects() ([]project, error) {
	var result []project
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: *gitlabEntriesPerPage},
		Visibility:  pGitlabVisibilityValue(gitlab.PublicVisibility),
	}
	for i := 1; *gitlabMaximumNumberOfEntries < 0 || i < *gitlabMaximumNumberOfEntries; {
		groupProjects, resp, err := instance.client.Groups.ListGroupProjects(3460920, opt)
		if err != nil {
			return result, fmt.Errorf("cannot search for users: %v", err)
		}
		for _, groupProject := range groupProjects {
			if *gitlabMaximumNumberOfEntries > 0 && i > *gitlabMaximumNumberOfEntries {
				break
			}
			if !groupProject.Archived {
				if project, err := instance.groupProjectToProject(*groupProject); err != nil {
					return nil, fmt.Errorf("cannot get details of group project '%s': %w", groupProject.Name, err)
				} else {
					result = append(result, project)
				}
				i++
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return result, nil
}

func (instance *gitlabClientRetrieveTask) detailsOfGroupProject(input gitlab.Project) (gitlab.Project, error) {
	if result, _, err := instance.client.Projects.GetProject(input.ID, nil); err != nil {
		return gitlab.Project{}, fmt.Errorf("cannot get details of GitLab repository echocat/%s(%d): %v", input.Name, input.ID, err)
	} else {
		return *result, nil
	}
}

func (instance *gitlabClientRetrieveTask) languagesOfGroupProject(input gitlab.Project) (gitlab.ProjectLanguages, error) {
	if result, _, err := instance.client.Projects.GetProjectLanguages(input.ID, nil); err != nil {
		return gitlab.ProjectLanguages{}, fmt.Errorf("cannot get languages of GitLab repository echocat/%s(%d): %v", input.Name, input.ID, err)
	} else {
		return *result, nil
	}
}

func (instance *gitlabClientRetrieveTask) languageOfGroupProject(input gitlab.Project) (result string, err error) {
	languages, err := instance.languagesOfGroupProject(input)
	if err != nil {
		return "", err
	}
	var maxRating float32 = 0.0
	for candidate, rating := range languages {
		if rating > maxRating {
			maxRating = rating
			result = candidate
		}
	}
	return
}

func (instance *gitlabClientRetrieveTask) groupProjectToProject(repo gitlab.Project) (project, error) {
	detailed, err := instance.detailsOfGroupProject(repo)
	if err != nil {
		return project{}, err
	}
	language, err := instance.languageOfGroupProject(repo)
	if err != nil {
		return project{}, err
	}
	name := detailed.Path
	fullname := detailed.Name
	if len(fullname) == 0 {
		fullname = name
	}
	issuesUrl := ""
	if detailed.IssuesEnabled {
		issuesUrl = detailed.WebURL + "/issues"
	}
	wikiUrl := ""
	if detailed.WikiEnabled {
		wikiUrl = detailed.WebURL + "/wiki"
	}
	pullRequestsUrl := ""
	if detailed.MergeRequestsEnabled {
		pullRequestsUrl = detailed.WebURL + "/-/merge_requests"
	}
	forksUrl := detailed.WebURL + "/forks"
	createForkUrl := detailed.WebURL + "/forks/new"
	starsUrl := detailed.WebURL + "/-/starrers"

	imageAsset := ""
	if detailed.AvatarURL != "" {
		r, err := instance.assetClient.retrieve(detailed.AvatarURL)
		if err != nil {
			return project{}, err
		}
		imageAsset = r
	}

	return project{
		Type:               "repository:git:gitlab",
		Origin:             "gitlab",
		Fullname:           fullname,
		Name:               name,
		Description:        pNonEmptyString(detailed.Description),
		DefaultBranch:      pNonEmptyString(detailed.DefaultBranch),
		Language:           pNonEmptyString(language),
		HomepageUrl:        pNonEmptyString(detailed.WebURL),
		ImageAsset:         pNonEmptyString(imageAsset),
		ProfileUrl:         detailed.WebURL,
		HttpCloneUrl:       pNonEmptyString(detailed.HTTPURLToRepo),
		SshCloneUrl:        pNonEmptyString(detailed.SSHURLToRepo),
		IssuesUrl:          pNonEmptyString(issuesUrl),
		WikiUrl:            pNonEmptyString(wikiUrl),
		PullRequestsUrl:    pNonEmptyString(pullRequestsUrl),
		ForksUrl:           &forksUrl,
		CreateForkUrl:      &createForkUrl,
		StarsUrl:           &starsUrl,
		NumberOfForks:      pUint32(uint32(detailed.ForksCount)),
		NumberOfOpenIssues: pUint32(uint32(detailed.OpenIssuesCount)),
		NumberOfStars:      pUint32(uint32(detailed.StarCount)),
		CreatedAt:          pTime(*detailed.CreatedAt),
		UpdatedAt:          pTime(*detailed.LastActivityAt),
	}, nil
}

func (instance *gitlabClient) newClient(_ context.Context) (*gitlab.Client, error) {
	return gitlab.NewClient(instance.accessToken)
}
