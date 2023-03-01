package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
	"net/http"
)

var (
	githubEntriesPerPage         = flag.Int("github-entriesPerPage", 50, "")
	githubMaximumNumberOfEntries = flag.Int("github-maximumNumberOfEntries", -1, "")
	githubAccessToken            = flag.String("githubAccessToken", "", "Github accessToken to access the API.")
)

type githubClient struct {
	accessToken string
	assetClient *assetClient
}

type githubClientRetrieveTask struct {
	*githubClient

	client *github.Client
	ctx    context.Context
}

func newGithubClient(assetClient *assetClient) *githubClient {
	return &githubClient{
		assetClient: assetClient,
		accessToken: *githubAccessToken,
	}
}

func (instance *githubClient) retrieveOrganization() (organization, error) {
	ctx := context.Background()

	task := githubClientRetrieveTask{
		githubClient: instance,
		client:       instance.newClient(ctx),
		ctx:          ctx,
	}

	return task.execute()
}

func (instance *githubClientRetrieveTask) execute() (organization, error) {
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

func (instance *githubClientRetrieveTask) retrieveMembers() ([]member, error) {
	var result []member
	opt := &github.ListMembersOptions{
		ListOptions: github.ListOptions{PerPage: *githubEntriesPerPage},
		PublicOnly:  true,
	}
	for i := 1; *githubMaximumNumberOfEntries < 0 || i < *githubMaximumNumberOfEntries; {
		users, resp, err := instance.client.Organizations.ListMembers(instance.ctx, "echocat", opt)
		if err != nil {
			return result, fmt.Errorf("cannot search for users: %v", err)
		}
		for _, user := range users {
			if *githubMaximumNumberOfEntries > 0 && i > *githubMaximumNumberOfEntries {
				break
			}
			if member, err := instance.userToMember(*user); err != nil {
				return nil, fmt.Errorf("cannot get details for user '%s': %w", *user.Name, err)
			} else {
				result = append(result, member)
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

func (instance *githubClientRetrieveTask) detailsOfUser(input github.User) (github.User, error) {
	if result, _, err := instance.client.Users.GetByID(instance.ctx, input.GetID()); err != nil {
		return github.User{}, fmt.Errorf("cannot get details of GitHub user %s(%d): %v", input.GetName(), input.GetID(), err)
	} else {
		return *result, nil
	}
}

func (instance *githubClientRetrieveTask) userToMember(repo github.User) (member, error) {
	if detailed, err := instance.detailsOfUser(repo); err != nil {
		return member{}, err
	} else {
		name := detailed.GetLogin()
		fullname := detailed.GetName()
		if len(fullname) == 0 {
			fullname = name
		}
		homepage := detailed.GetBlog()
		if len(homepage) == 0 {
			homepage = detailed.GetHTMLURL()
		}

		imageAsset := ""
		if avatarUrl := detailed.GetAvatarURL(); avatarUrl != "" {
			r, err := instance.assetClient.retrieve(avatarUrl)
			if err != nil {
				return member{}, err
			}
			imageAsset = r
		}
		avatarUrl := detailed.GetAvatarURL()
		if len(avatarUrl) == 0 {
			avatarUrl = "https://www.gravatar.com/avatar/00000000000000000000000000000000"
		}

		return member{
			Type:        "user:github",
			Fullname:    fullname,
			Name:        name,
			Email:       pString(detailed.GetEmail()),
			ImageAsset:  imageAsset,
			ProfileUrl:  detailed.GetHTMLURL(),
			Bio:         pString(detailed.GetBio()),
			Location:    pString(detailed.GetLocation()),
			Company:     pString(detailed.GetCompany()),
			HomepageUrl: &homepage,
			CreatedAt:   pTime(detailed.GetCreatedAt().Time),
			UpdatedAt:   pTime(detailed.GetUpdatedAt().Time),
		}, nil
	}
}

func (instance *githubClientRetrieveTask) retrieveProjects() ([]project, error) {
	var result []project
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: *githubEntriesPerPage},
		Visibility:  "public",
	}
	for i := 1; *githubMaximumNumberOfEntries < 0 || i < *githubMaximumNumberOfEntries; {
		repos, resp, err := instance.client.Repositories.List(instance.ctx, "echocat", opt)
		if err != nil {
			return result, fmt.Errorf("cannot search for users: %v", err)
		}
		for _, repo := range repos {
			if *githubMaximumNumberOfEntries > 0 && i > *githubMaximumNumberOfEntries {
				break
			}
			if !repo.GetArchived() {
				if project, err := instance.repoToProject(*repo); err != nil {
					return nil, fmt.Errorf("cannot get details of project '%s': %w", *repo.Name, err)
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

func (instance *githubClientRetrieveTask) detailsOfProject(input github.Repository) (github.Repository, error) {
	if result, _, err := instance.client.Repositories.GetByID(instance.ctx, input.GetID()); err != nil {
		return github.Repository{}, fmt.Errorf("cannot get details of GitHub repository %s/%s(%d): %v", input.GetOwner().GetLogin(), input.GetName(), input.GetID(), err)
	} else {
		return *result, nil
	}
}

func (instance *githubClientRetrieveTask) repoToProject(repo github.Repository) (project, error) {
	if detailed, err := instance.detailsOfProject(repo); err != nil {
		return project{}, err
	} else {
		name := detailed.GetName()
		fullname := detailed.GetFullName()
		if len(fullname) == 0 {
			fullname = name
		}
		homepage := detailed.GetHomepage()
		if len(homepage) == 0 {
			homepage = detailed.GetHTMLURL()
		}
		issuesUrl := ""
		if detailed.GetHasIssues() {
			issuesUrl = detailed.GetHTMLURL() + "/issues"
		}
		wikiUrl := ""
		if detailed.GetHasWiki() {
			wikiUrl = detailed.GetHTMLURL() + "/wiki"
		}
		pullRequestsUrl := detailed.GetHTMLURL() + "/pulls"
		forksUrl := detailed.GetHTMLURL() + "/network"
		createForkUrl := detailed.GetHTMLURL() + "/fork"
		starsUrl := detailed.GetHTMLURL() + "/stargazers"
		watchersUrl := detailed.GetHTMLURL() + "/watchers"

		return project{
			Type:               "repository:git:github",
			Origin:             "github",
			Fullname:           fullname,
			Name:               name,
			Description:        pString(detailed.GetDescription()),
			DefaultBranch:      pString(detailed.GetDefaultBranch()),
			Language:           pString(detailed.GetLanguage()),
			HomepageUrl:        &homepage,
			ProfileUrl:         detailed.GetHTMLURL(),
			HttpCloneUrl:       pString(detailed.GetCloneURL()),
			SshCloneUrl:        pString(detailed.GetSSHURL()),
			IssuesUrl:          pNonEmptyString(issuesUrl),
			WikiUrl:            pNonEmptyString(wikiUrl),
			ForksUrl:           &forksUrl,
			PullRequestsUrl:    &pullRequestsUrl,
			CreateForkUrl:      &createForkUrl,
			StarsUrl:           &starsUrl,
			WatchersUrl:        &watchersUrl,
			NumberOfForks:      pUint32(uint32(detailed.GetForksCount())),
			NumberOfOpenIssues: pUint32(uint32(detailed.GetOpenIssuesCount())),
			NumberOfStars:      pUint32(uint32(detailed.GetStargazersCount())),
			NumberOfWatchers:   pUint32(uint32(detailed.GetWatchersCount())),
			CreatedAt:          pTime(detailed.GetCreatedAt().Time),
			UpdatedAt:          pTime(detailed.GetPushedAt().Time),
		}, nil
	}
}

func (instance *githubClient) newClient(ctx context.Context) *github.Client {
	var httpClient *http.Client
	if len(instance.accessToken) > 0 {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: instance.accessToken},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	}
	return github.NewClient(httpClient)
}
