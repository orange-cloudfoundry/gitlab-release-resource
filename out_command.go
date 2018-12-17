package resource

import (
	"fmt"
	"io"
	"io/ioutil"
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

func (c *OutCommand) Run(sourceDir string, request OutRequest) (OutResponse, error) {
	params := request.Params

	name, err := c.fileContents(filepath.Join(sourceDir, request.Params.NamePath))
	if err != nil {
		return OutResponse{}, err
	}

	tag, err := c.fileContents(filepath.Join(sourceDir, request.Params.TagPath))
	if err != nil {
		return OutResponse{}, err
	}

	tag = request.Params.TagPrefix + tag

	targetCommitish, err = c.fileContents(filepath.Join(sourceDir, request.Params.CommitishPath))
	if err != nil {
		return OutResponse{}, err
	}

	var body string
	bodySpecified := false
	if request.Params.BodyPath != "" {
		bodySpecified = true

		body, err = c.fileContents(filepath.Join(sourceDir, request.Params.BodyPath))
		if err != nil {
			return OutResponse{}, err
		}
	}

	release := &gitlab.RepositoryRelease{
		Name:            gitlab.String(name),
		TagName:         gitlab.String(tag),
		Body:            gitlab.String(body),
		TargetCommitish: gitlab.String(targetCommitish),
	}

	tagExists := false
	existingTag, err := c.gitlab.GetTag(tag)
	if err != nil {
		//TODO: improve the check to be based on the specific error
		tagExists = true
	}

	// create the tag first, as next sections assume the tag exists
	if !tagExists {
		tag, err := c.gitlab.CreateTag(targetCommitish, tag)
		if err != nil {
			return OutResponse{}, err
		}
	}

	// create a new release
	_, err = c.gitlab.CreateRelease(tag, "Auto-generated from Concourse GitLab Release Resource")
	if err != nil {
		// if 409 error occurs, this means the release already existed, so just skip to the next section (update the release)
		if err.Error() != "release already exists" {
			return OutResponse{}, err
		}
	}

	// upload files
	var fileLinks []string
	for _, fileGlob := range params.Globs {
		matches, err := filepath.Glob(filepath.Join(sourceDir, fileGlob))
		if err != nil {
			return OutResponse{}, err
		}

		if len(matches) == 0 {
			return OutResponse{}, fmt.Errorf("could not find file that matches glob '%s'", fileGlob)
		}

		for _, filePath := range matches {
			projectFile, err := c.UploadProjectFile(filePath)
			if err != nil {
				return OutResponse{}, err
			}
			fileLinks = append(fileLinks, projectFile.Markdown)
		}
	}

	// update the release
	release, err = c.gitlab.UpdateRelease(tag, fileLinks.Join("\n"))

	return OutResponse{
		Version:  versionFromRelease(release),
		Metadata: metadataFromRelease(release, ""),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
