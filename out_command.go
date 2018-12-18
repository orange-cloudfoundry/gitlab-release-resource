package resource

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
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

	// name, err := c.fileContents(filepath.Join(sourceDir, request.Params.NamePath))
	// if err != nil {
	// 	return OutResponse{}, err
	// }

	tag_name, err := c.fileContents(filepath.Join(sourceDir, request.Params.TagPath))
	if err != nil {
		return OutResponse{}, err
	}

	tag_name = request.Params.TagPrefix + tag_name

	targetCommitish, err := c.fileContents(filepath.Join(sourceDir, request.Params.CommitishPath))
	if err != nil {
		return OutResponse{}, err
	}

	// if request.Params.BodyPath != "" {
	// 	_, err := c.fileContents(filepath.Join(sourceDir, request.Params.BodyPath))
	// 	if err != nil {
	// 		return OutResponse{}, err
	// 	}
	// }

	tagExists := true
	tag, err := c.gitlab.GetTag(tag_name)
	if err != nil {
		//TODO: improve the check to be based on the specific error
		tagExists = false
	}

	// create the tag first, as next sections assume the tag exists
	if !tagExists {
		tag, err = c.gitlab.CreateTag(targetCommitish, tag_name)
		if err != nil {
			return OutResponse{}, err
		}
	}

	// create a new release if it doesn't exist yet
	if tag.Release == nil {
		_, err = c.gitlab.CreateRelease(tag_name, "Auto-generated from Concourse GitLab Release Resource")
		if err != nil {
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
			projectFile, err := c.gitlab.UploadProjectFile(filePath)
			if err != nil {
				return OutResponse{}, err
			}
			fileLinks = append(fileLinks, projectFile.Markdown)
		}
	}

	// update the release
	_, err = c.gitlab.UpdateRelease(tag_name, strings.Join(fileLinks, "\n"))
	if err != nil {
		return OutResponse{}, errors.New("could not get saved tag")
	}

	return OutResponse{
		Version:  Version{Tag: tag_name},
		Metadata: metadataFromTag(tag),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
