package resource

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"

	"context"

	"github.com/xanzy/go-gitlab"
)

var (
	NotFound = errors.New("object not found")
)

//go:generate counterfeiter . GitLab

type GitLab interface {
	ListTags() ([]*gitlab.Tag, error)
	ListTagsUntil(tag_name string) ([]*gitlab.Tag, error)
	ListReleases() ([]*gitlab.Release, error)
	GetRelease(tag_name string) (*gitlab.Release, error)
	GetTag(tag_name string) (*gitlab.Tag, error)
	CreateTag(tag_name string, ref string) (*gitlab.Tag, error)
	CreateRelease(name string, tag string, description *string) (*gitlab.Release, error)
	UpdateRelease(name string, tag string, description *string) (*gitlab.Release, error)

	UploadProjectFile(file string) (*gitlab.ProjectFile, error)
	DownloadProjectFile(url, file string) error

	GetReleaseLinks(tag string) ([]*gitlab.ReleaseLink, error)
	CreateReleaseLink(tag string, name string, url string) (*gitlab.ReleaseLink, error)
	DeleteReleaseLink(tag string, links *gitlab.ReleaseLink) (error)
}

const (
	defaultBaseURL = "https://gitlab.com/"
)

type GitlabClient struct {
	client *gitlab.Client

	accessToken string
	repository  string
}

func NewGitLabClient(source Source) (*GitlabClient, error) {
	var httpClient = &http.Client{}
	var ctx = context.TODO()

	if source.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}
	httpClientOpt := gitlab.WithHTTPClient(httpClient)

	baseURLOpt := gitlab.WithBaseURL(defaultBaseURL)
	if source.GitLabAPIURL != "" {
		var err error
		baseUrl, err := url.Parse(source.GitLabAPIURL)
		if err != nil {
			return nil, err
		}
		baseURLOpt = gitlab.WithBaseURL(baseUrl.String())
	}

	client, err := gitlab.NewClient(source.AccessToken, httpClientOpt, baseURLOpt)
	if err != nil {
		return nil, err
	}

	return &GitlabClient{
		client:      client,
		repository:  source.Repository,
		accessToken: source.AccessToken,
	}, nil
}

func (g *GitlabClient) ListTags() ([]*gitlab.Tag, error) {
	var allTags []*gitlab.Tag

	opt := &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		OrderBy: gitlab.String("updated"),
		Sort:    gitlab.String("desc"),
	}

	for {
		tags, res, err := g.client.Tags.ListTags(g.repository, opt)
		if err != nil {
			return []*gitlab.Tag{}, err
		}

		allTags = append(allTags, tags...)

		if opt.Page >= res.TotalPages {
			break
		}

		opt.Page = res.NextPage
	}

	return allTags, nil
}



func (g *GitlabClient) ListReleases() ([]*gitlab.Release, error) {
	var allReleases []*gitlab.Release
	opt := &gitlab.ListReleasesOptions{
		PerPage: 100,
		Page:    1,
	}

	for {
		releases, res, err := g.client.Releases.ListReleases(g.repository, opt)
		if err != nil {
			return []*gitlab.Release{}, err
		}
		allReleases = append(allReleases, releases...)

		if opt.Page >= res.TotalPages {
			break
		}
		opt.Page = res.NextPage
	}

	return allReleases, nil
}

func (g *GitlabClient) ListTagsUntil(tag_name string) ([]*gitlab.Tag, error) {
	var allTags []*gitlab.Tag

	pageSize := 100

	opt := &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: pageSize,
			Page:    1,
		},
		OrderBy: gitlab.String("updated"),
		Sort:    gitlab.String("desc"),
	}

	var foundTag *gitlab.Tag
	for {
		tags, res, err := g.client.Tags.ListTags(g.repository, opt)
		if err != nil {
			return []*gitlab.Tag{}, err
		}

		skipToNextPage := false
		for i, tag := range tags {
			// Some tags might have the same date - if they all have the same date, take
			// all of them
			if foundTag != nil {
				if foundTag.Commit.CommittedDate.Equal(*tag.Commit.CommittedDate) {
					allTags = append(allTags, tag)
					if i == (pageSize - 1) {
						skipToNextPage = true
						break
					} else {
						continue
					}
				} else {
					break
				}
			}

			if tag.Name == tag_name {
				allTags = append(allTags, tags[:i+1]...)
				foundTag = tag
				continue
			}
		}
		if skipToNextPage {
			if opt.Page >= res.TotalPages {
				break
			}

			opt.Page = res.NextPage
			continue
		}

		if foundTag != nil {
			break
		}

		allTags = append(allTags, tags...)

		if opt.Page >= res.TotalPages {
			break
		}

		opt.Page = res.NextPage
	}

	return allTags, nil
}

func (g *GitlabClient) GetTag(tag_name string) (*gitlab.Tag, error) {
	tag, resp, err := g.client.Tags.GetTag(g.repository, tag_name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, NotFound
		}
		return nil, err
	}

	defer resp.Body.Close()
	return tag, nil
}

func (g *GitlabClient) GetRelease(tag_name string) (*gitlab.Release, error) {
	release, resp, err := g.client.Releases.GetRelease(g.repository, tag_name)
	if err != nil {
		if resp == nil {
			return nil, err
		}
		switch resp.StatusCode {
		case http.StatusForbidden:
			return nil, NotFound
		case http.StatusNotFound:
			return nil, NotFound
		default:
			return nil, err
		}
	}

	return release, nil
}

func (g *GitlabClient) CreateTag(ref string, tag_name string) (*gitlab.Tag, error) {
	opt := &gitlab.CreateTagOptions{
		TagName: gitlab.String(tag_name),
		Ref:     gitlab.String(ref),
		Message: gitlab.String(tag_name),
	}

	tag, _, err := g.client.Tags.CreateTag(g.repository, opt)
	if err != nil {
		return &gitlab.Tag{}, err
	}

	return tag, nil
}

func (g *GitlabClient) CreateRelease(name string, tag string, description *string) (*gitlab.Release, error) {
	opt := &gitlab.CreateReleaseOptions{
		Name: gitlab.String(name),
		TagName: gitlab.String(tag),
		Description: description,
	}

	release, res, err := g.client.Releases.CreateRelease(g.repository, opt)
	if err != nil {
		return &gitlab.Release{}, err
	}

	// https://docs.gitlab.com/ce/api/tags.html#create-a-new-release
	// returns 409 if release already exists
	if res.StatusCode == http.StatusConflict {
		return nil, errors.New("release already exists")
	}

	return release, nil
}

func (g *GitlabClient) GetReleaseLinks(tag string) ([]*gitlab.ReleaseLink, error) {
	links := []*gitlab.ReleaseLink{}
	opt := &gitlab.ListReleaseLinksOptions{
		PerPage: 100,
		Page: 1,
	}
	for {
		items, resp, err := g.client.ReleaseLinks.ListReleaseLinks(g.repository, tag, opt)
		if err != nil {
			return nil, err
		}

		links = append(links, items...)
		if opt.Page >= resp.TotalPages {
			break
		}
		opt.Page = resp.NextPage
	}
	return links, nil
}

func (g *GitlabClient) DeleteReleaseLink(tag string, link *gitlab.ReleaseLink) (error) {
	_, _, err := g.client.ReleaseLinks.DeleteReleaseLink(g.repository, tag, link.ID)
	if err != nil {
		return err
	}
	return nil
}


func (g *GitlabClient) CreateReleaseLink(tag string, name string, url string) (*gitlab.ReleaseLink, error) {
	opt := &gitlab.CreateReleaseLinkOptions{
		Name: gitlab.String(name),
		URL:  gitlab.String(url),
	}
	link, _, err := g.client.ReleaseLinks.CreateReleaseLink(g.repository, tag, opt)
	if err != nil {
		return nil, err
	}

	return link, nil
}

func (g *GitlabClient) UpdateRelease(name string, tag string, description *string) (*gitlab.Release, error) {
	opt := &gitlab.UpdateReleaseOptions{
		Name: gitlab.String(name),
		Description: description,
	}

	release, _, err := g.client.Releases.UpdateRelease(g.repository, tag, opt)
	if err != nil {
		return &gitlab.Release{}, err
	}

	return release, nil
}

func (g *GitlabClient) UploadProjectFile(file string) (*gitlab.ProjectFile, error) {
	projectFile, _, err := g.client.Projects.UploadFile(g.repository, file)
	if err != nil {
		return &gitlab.ProjectFile{}, err
	}

	return projectFile, nil
}

func (g *GitlabClient) DownloadProjectFile(fileURL, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// e.g. (baseURL) + (group/project) + (/uploads/hash/filename)
	filePathRef, err := url.Parse(fileURL)
	if err != nil {
		return err
	}

	// https://gitlab.com/gitlab-org/gitlab-ce/issues/51447
	nonApiUrl := strings.Replace(filePathRef.String(), "/api/v4", "", 1)
	filePathRef, err = url.Parse(nonApiUrl)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", filePathRef.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Private-Token", g.accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file `%s`: HTTP status %d", filepath.Base(destPath), resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
