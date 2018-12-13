package resource

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2"

	"context"

	"github.com/xanzy/go-gitlab"
)

//go:generate counterfeiter . GitHub

type gitlab interface {
	ListReleases() ([]*gitlab.RepositoryRelease, error)
	GetReleaseByTag(tag string) (*gitlab.RepositoryRelease, error)
	GetRelease(id int) (*gitlab.RepositoryRelease, error)
	CreateRelease(release gitlab.RepositoryRelease) (*gitlab.RepositoryRelease, error)
	UpdateRelease(release gitlab.RepositoryRelease) (*gitlab.RepositoryRelease, error)

	ListReleaseAssets(release gitlab.RepositoryRelease) ([]*gitlab.ReleaseAsset, error)
	UploadReleaseAsset(release gitlab.RepositoryRelease, name string, file *os.File) error
	DeleteReleaseAsset(asset gitlab.ReleaseAsset) error
	DownloadReleaseAsset(asset gitlab.ReleaseAsset) (io.ReadCloser, error)

	GetTarballLink(tag string) (*url.URL, error)
	GetZipballLink(tag string) (*url.URL, error)
	GetRef(tag string) (*gitlab.Reference, error)
}

type gitlabClient struct {
	client *gitlab.Client

	owner      string
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

	if source.AccessToken != "" {
		var err error
		httpClient, err = oauthClient(ctx, source)
		if err != nil {
			return nil, err
		}
	}

	client := gitlab.NewClient(httpClient)

	if source.gitlabAPIURL != "" {
		var err error
		client.BaseURL, err = url.Parse(source.gitlabAPIURL)
		if err != nil {
			return nil, err
		}

		client.UploadURL, err = url.Parse(source.gitlabAPIURL)
		if err != nil {
			return nil, err
		}
	}

	if source.gitlabUploadsURL != "" {
		var err error
		client.UploadURL, err = url.Parse(source.gitlabUploadsURL)
		if err != nil {
			return nil, err
		}
	}

	owner := source.Owner
	if source.User != "" {
		owner = source.User
	}

	return &gitlabClient{
		client:     client,
		owner:      owner,
		repository: source.Repository,
	}, nil
}

func (g *gitlabClient) ListReleases() ([]*gitlab.RepositoryRelease, error) {
	releases, res, err := g.client.Repositories.ListReleases(context.TODO(), g.owner, g.repository, nil)
	if err != nil {
		return []*gitlab.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (g *gitlabClient) GetReleaseByTag(tag string) (*gitlab.RepositoryRelease, error) {
	release, res, err := g.client.Repositories.GetReleaseByTag(context.TODO(), g.owner, g.repository, tag)
	if err != nil {
		return &gitlab.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *gitlabClient) GetRelease(id int) (*gitlab.RepositoryRelease, error) {
	release, res, err := g.client.Repositories.GetRelease(context.TODO(), g.owner, g.repository, id)
	if err != nil {
		return &gitlab.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *gitlabClient) CreateRelease(release gitlab.RepositoryRelease) (*gitlab.RepositoryRelease, error) {
	createdRelease, res, err := g.client.Repositories.CreateRelease(context.TODO(), g.owner, g.repository, &release)
	if err != nil {
		return &gitlab.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return createdRelease, nil
}

func (g *gitlabClient) UpdateRelease(release gitlab.RepositoryRelease) (*gitlab.RepositoryRelease, error) {
	if release.ID == nil {
		return nil, errors.New("release did not have an ID: has it been saved yet?")
	}

	updatedRelease, res, err := g.client.Repositories.EditRelease(context.TODO(), g.owner, g.repository, *release.ID, &release)
	if err != nil {
		return &gitlab.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return updatedRelease, nil
}

func (g *gitlabClient) ListReleaseAssets(release gitlab.RepositoryRelease) ([]*gitlab.ReleaseAsset, error) {
	assets, res, err := g.client.Repositories.ListReleaseAssets(context.TODO(), g.owner, g.repository, *release.ID, nil)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return assets, nil
}

func (g *gitlabClient) UploadReleaseAsset(release gitlab.RepositoryRelease, name string, file *os.File) error {
	_, res, err := g.client.Repositories.UploadReleaseAsset(
		context.TODO(),
		g.owner,
		g.repository,
		*release.ID,
		&gitlab.UploadOptions{
			Name: name,
		},
		file,
	)
	if err != nil {
		return err
	}

	return res.Body.Close()
}

func (g *gitlabClient) DeleteReleaseAsset(asset gitlab.ReleaseAsset) error {
	res, err := g.client.Repositories.DeleteReleaseAsset(context.TODO(), g.owner, g.repository, *asset.ID)
	if err != nil {
		return err
	}

	return res.Body.Close()
}

func (g *gitlabClient) DownloadReleaseAsset(asset gitlab.ReleaseAsset) (io.ReadCloser, error) {
	res, redir, err := g.client.Repositories.DownloadReleaseAsset(context.TODO(), g.owner, g.repository, *asset.ID)
	if err != nil {
		return nil, err
	}

	if redir != "" {
		resp, err := http.Get(redir)
		if err != nil {
			return nil, err
		}

		return resp.Body, nil
	}

	return res, err
}

func (g *gitlabClient) GetTarballLink(tag string) (*url.URL, error) {
	opt := &gitlab.RepositoryContentGetOptions{Ref: tag}
	u, res, err := g.client.Repositories.GetArchiveLink(context.TODO(), g.owner, g.repository, gitlab.Tarball, opt)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func (g *gitlabClient) GetZipballLink(tag string) (*url.URL, error) {
	opt := &gitlab.RepositoryContentGetOptions{Ref: tag}
	u, res, err := g.client.Repositories.GetArchiveLink(context.TODO(), g.owner, g.repository, gitlab.Zipball, opt)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func (g *gitlabClient) GetRef(tag string) (*gitlab.Reference, error) {
	ref, res, err := g.client.Git.GetRef(context.TODO(), g.owner, g.repository, "tags/"+tag)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return ref, nil
}

func oauthClient(ctx context.Context, source Source) (*http.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: source.AccessToken,
	})

	oauthClient := oauth2.NewClient(ctx, ts)

	gitlabHTTPClient := &http.Client{
		Transport: oauthClient.Transport,
	}

	return gitlabHTTPClient, nil
}
