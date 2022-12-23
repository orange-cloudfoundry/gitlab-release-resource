package resource

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xanzy/go-gitlab"
)

type OutCommand struct {
	gitlab GitLab
	writer io.Writer
}

func NewOutCommand(gitlab GitLab, writer io.Writer) *OutCommand {
	return &OutCommand{
		gitlab: gitlab,
		writer: writer,
	}
}

func (c *OutCommand) ensureRelease(name string, tag string, body *string) (*gitlab.Release, error) {
	_, err := c.gitlab.GetRelease(tag)
	if err != nil {
		if !errors.Is(err, NotFound) {
			return nil, err
		}
		return c.gitlab.CreateRelease(name, tag, body)
	}
	return c.gitlab.UpdateRelease(name, tag, body)
}

func (c *OutCommand) ensureTag(tag string, commitishPath string) (*gitlab.Tag, error) {
	t, err := c.gitlab.GetTag(tag)
	if err != nil {
		if !errors.Is(err, NotFound) {
			return nil, err
		}
		commitish, err := c.fileContents(commitishPath)
		if err != nil {
			return nil, err
		}
		return c.gitlab.CreateTag(tag, commitish)
	}
	return t, nil
}

func (c *OutCommand) overwriteReleaseLinks(tag string, filePaths []string, req OutRequest) error {
	links, err := c.gitlab.GetReleaseLinks(tag)
	if err != nil {
		return err
	}

	for _, link := range links {
		if err := c.gitlab.DeleteReleaseLink(tag, link); err != nil {
			return err
		}
	}

	for _, file := range filePaths {
		uploadedFile, err := c.gitlab.UploadProjectFile(file)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/%s/%s", req.Source.GitLabAPIURL, req.Source.Repository, uploadedFile.URL)
		if _, err := c.gitlab.CreateReleaseLink(tag, filepath.Base(file), url); err != nil {
			return err
		}
	}
	return nil
}

func (c *OutCommand) Run(sourceDir string, request OutRequest) (OutResponse, error) {
	var (
		body *string
	)
	params := request.Params

	tag_name, err := c.fileContents(filepath.Join(sourceDir, params.TagPath))
	if err != nil {
		return OutResponse{}, err
	}
	tag_name = request.Params.TagPrefix + tag_name

	if params.NamePath == "" {
		params.NamePath = params.TagPath
	}
	name, err := c.fileContents(filepath.Join(sourceDir, params.NamePath))
	if err != nil {
		return OutResponse{}, err
	}

	if params.BodyPath != "" {
		bodyVal, err := c.fileContents(filepath.Join(sourceDir, params.BodyPath))
		if err != nil {
			return OutResponse{}, err
		}
		body = &bodyVal
	}

	// ensure the tag exists, create from commitish if needed
	_, err = c.ensureTag(tag_name, filepath.Join(sourceDir, params.CommitishPath))
	if err != nil {
		if err != nil {
			return OutResponse{}, err
		}
	}

	// ensure release exists, create from name, tag and body if needed
	r, err := c.ensureRelease(name, tag_name, body)
	if err != nil {
		return OutResponse{}, err
	}

	filePaths := []string{}
	for _, glob := range params.Globs {
		matches, err := filepath.Glob(filepath.Join(sourceDir, glob))
		if err != nil {
			return OutResponse{}, err
		}

		if len(matches) == 0 {
			return OutResponse{}, fmt.Errorf("could not find file that matches glob '%s'", glob)
		}
		filePaths = append(filePaths, matches...)
	}
	if err := c.overwriteReleaseLinks(tag_name, filePaths, request); err != nil {
		return OutResponse{}, err
	}

	return OutResponse{
		Version:  versionFromRelease(r),
		Metadata: metadataFromRelease(r),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
