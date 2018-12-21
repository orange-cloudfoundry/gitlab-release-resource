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

//go:generate counterfeiter . GitLab

type GitLab interface {
	ListTags() ([]*gitlab.Tag, error)
	ListTagsUntil(tag_name string) ([]*gitlab.Tag, error)
	GetTag(tag_name string) (*gitlab.Tag, error)
	CreateTag(tag_name string, ref string) (*gitlab.Tag, error)
	CreateRelease(tag_name string, description string) (*gitlab.Release, error)
	UpdateRelease(tag_name string, description string) (*gitlab.Release, error)
	UploadProjectFile(file string) (*gitlab.ProjectFile, error)
	DownloadProjectFile(url, file string) error
}

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

	client := gitlab.NewClient(httpClient, source.AccessToken)

	if source.GitLabAPIURL != "" {
		var err error
		baseUrl, err := url.Parse(source.GitLabAPIURL)
		if err != nil {
			return nil, err
		}
		client.SetBaseURL(baseUrl.String())
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

func (g *GitlabClient) CreateTag(ref string, tag_name string) (*gitlab.Tag, error) {
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

func (g *GitlabClient) CreateRelease(tag_name string, description string) (*gitlab.Release, error) {
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

func (g *GitlabClient) UpdateRelease(tag_name string, description string) (*gitlab.Release, error) {
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

func (g *GitlabClient) UploadProjectFile(file string) (*gitlab.ProjectFile, error) {
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

func (g *GitlabClient) DownloadProjectFile(filePath, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// e.g. (group/project) + (/uploads/hash/filename)
	filePathRef, err := url.Parse(g.repository + filePath)
	if err != nil {
		return err
	}

	// e.g. (https://gitlab-instance/api/v4) + (/group/project/uploads/hash/filename)
	projectFileUrl := g.client.BaseURL().ResolveReference(filePathRef)

	// https://gitlab.com/gitlab-org/gitlab-ce/issues/51447
	nonApiUrl := strings.Replace(projectFileUrl.String(), "/api/v4", "", 1)
	projectFileUrl, err = url.Parse(nonApiUrl)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", projectFileUrl.String(), nil)
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
