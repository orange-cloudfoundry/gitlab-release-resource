package resource

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"context"

	"github.com/xanzy/go-gitlab"
)

//go:generate counterfeiter . GitLab

type GitLab interface {
	ListTags() ([]*gitlab.Tag, error)
	ListTagsUntil(tag_name string) ([]*gitlab.Tag, error)
	GetTag(tag_name string) (*gitlab.Tag, error)
	CreateTag(tag_name string, ref string) (*gitlab.Tag, error)
	CreateRelease(tag_name string, description string) (*gitlab.Release, error)
	UpdateRelease(description string) (*gitlab.Release, error)
	UploadProjectFile(file string) (*gitlab.ProjectFile, error)
}

type gitlabClient struct {
	client *gitlab.Client

	repository string
}

func NewGitlabClient(source Source) (*gitlabClient, error) {
	var httpClient = &http.Client{}
	var ctx = context.TODO()

	if source.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	client := gitlab.NewClient(httpClient, source.AccessToken)

	if source.GitlabAPIURL != "" {
		var err error
		baseUrl, err := url.Parse(source.GitlabAPIURL)
		if err != nil {
			return nil, err
		}
		client.SetBaseURL(baseUrl.String())
	}

	return &gitlabClient{
		client:     client,
		repository: source.Repository,
	}, nil
}

func (g *gitlabClient) ListTags() ([]*gitlab.Tag, error) {
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

		if opt.Page >= res.TotalPages {
			break
		}

		opt.Page = res.NextPage

		allTags = append(allTags, tags...)
	}

	return allTags, nil
}

func (g *gitlabClient) ListTagsUntil(tag_name string) ([]*gitlab.Tag, error) {
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

		if opt.Page >= res.TotalPages {
			break
		}

		for i, tag := range tags {
			if tag.Name == tag_name {
				allTags = append(allTags, tags[:i]...)
				break
			}
		}

		opt.Page = res.NextPage
		allTags = append(allTags, tags...)
	}

	return allTags, nil
}

func (g *gitlabClient) GetTag(tag_name string) (*gitlab.Tag, error) {
	tag, res, err := g.client.Tags.GetTag(g.repository, tag_name)
	if err != nil {
		return &gitlab.Tag{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (g *gitlabClient) CreateTag(ref string, tag_name string) (*gitlab.Tag, error) {
	opt := &gitlab.CreateTagOptions{
		TagName: gitlab.String(tag_name),
		Ref:     gitlab.String(ref),
		Message: gitlab.String(tag_name),
	}

	tag, res, err := g.client.Tags.CreateTag(g.repository, opt)
	if err != nil {
		return &gitlab.Tag{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func (g *gitlabClient) CreateRelease(tag_name string, description string) (*gitlab.Release, error) {
	opt := &gitlab.CreateReleaseOptions{
		Description: gitlab.String(description),
	}

	release, res, err := g.client.Tags.CreateRelease(g.repository, tag_name, opt)
	if err != nil {
		return &gitlab.Release{}, err
	}

	// https://docs.gitlab.com/ce/api/tags.html#create-a-new-release
	// returns 409 if release already exists
	if res.StatusCode == http.StatusConflict {
		return nil, errors.New("release already exists")
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *gitlabClient) UpdateRelease(tag_name string, description string) (*gitlab.Release, error) {
	opt := &gitlab.UpdateReleaseOptions{
		Description: gitlab.String(description),
	}

	release, res, err := g.client.Tags.UpdateRelease(g.repository, tag_name, opt)
	if err != nil {
		return &gitlab.Release{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *gitlabClient) UploadProjectFile(file string) (*gitlab.ProjectFile, error) {
	projectFile, res, err := g.client.Projects.UploadFile(g.repository, file)
	if err != nil {
		return &gitlab.ProjectFile{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return projectFile, nil
}
