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

	tag, err := c.fileContents(filepath.Join(sourceDir, request.Params.TagPath))
	if err != nil {
		return OutResponse{}, err
	}

	tag = request.Params.TagPrefix + tag

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

	tagExists := false
	_, err = c.gitlab.GetTag(tag)
	if err != nil {
		//TODO: improve the check to be based on the specific error
		tagExists = true
	}

	// create the tag first, as next sections assume the tag exists
	if !tagExists {
		_, err := c.gitlab.CreateTag(targetCommitish, tag)
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
			projectFile, err := c.gitlab.UploadProjectFile(filePath)
			if err != nil {
				return OutResponse{}, err
			}
			fileLinks = append(fileLinks, projectFile.Markdown)
		}
	}

	// update the release
	_, err = c.gitlab.UpdateRelease(tag, strings.Join(fileLinks, "\n"))

	// get tag
	savedTag, err := c.gitlab.GetTag(tag)
	if err != nil {
		return OutResponse{}, errors.New("could not get saved tag")
	}

	return OutResponse{
		Version:  Version{Tag: tag},
		Metadata: metadataFromTag(savedTag),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
