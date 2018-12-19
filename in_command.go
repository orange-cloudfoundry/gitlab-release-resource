package resource

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/xanzy/go-gitlab"
)

type InCommand struct {
	gitlab GitLab
	writer io.Writer
}

type attachment struct {
	Name string
	URL  string
}

func NewInCommand(gitlab GitLab, writer io.Writer) *InCommand {
	return &InCommand{
		gitlab: gitlab,
		writer: writer,
	}
}

func (c *InCommand) Run(destDir string, request InRequest) (InResponse, error) {
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return InResponse{}, err
	}

	var foundTag *gitlab.Tag

	foundTag, err = c.gitlab.GetTag(request.Version.Tag)
	if err != nil {
		return InResponse{}, err
	}

	if foundTag == nil {
		return InResponse{}, errors.New("could not find tag")
	}

	tagPath := filepath.Join(destDir, "tag")
	err = ioutil.WriteFile(tagPath, []byte(foundTag.Name), 0644)
	if err != nil {
		return InResponse{}, err
	}

	versionParser, err := newVersionParser(request.Source.TagFilter)
	if err != nil {
		return InResponse{}, err
	}
	version := versionParser.parse(foundTag.Name)
	versionPath := filepath.Join(destDir, "version")
	err = ioutil.WriteFile(versionPath, []byte(version), 0644)
	if err != nil {
		return InResponse{}, err
	}

	commitPath := filepath.Join(destDir, "commit_sha")
	err = ioutil.WriteFile(commitPath, []byte(foundTag.Commit.ID), 0644)
	if err != nil {
		return InResponse{}, err
	}

	if foundTag.Release != nil && foundTag.Release.Description != "" {
		body := foundTag.Release.Description
		bodyPath := filepath.Join(destDir, "body")
		err = ioutil.WriteFile(bodyPath, []byte(body), 0644)
		if err != nil {
			return InResponse{}, err
		}
	} else {
		return InResponse{}, errors.New("release notes for the tag was empty")
	}

	attachments, err := c.getAttachments(foundTag.Release.Description)
	if err != nil {
		return InResponse{}, err
	}

	for _, attachment := range attachments {
		path := filepath.Join(destDir, attachment.Name)

		var matchFound bool
		if len(request.Params.Globs) == 0 {
			matchFound = true
		} else {
			for _, glob := range request.Params.Globs {
				matches, err := filepath.Match(glob, attachment.Name)
				if err != nil {
					return InResponse{}, err
				}

				if matches {
					matchFound = true
					break
				}
			}
		}

		if !matchFound {
			continue
		}

		err := c.gitlab.DownloadProjectFile(attachment.URL, path)
		if err != nil {
			return InResponse{}, err
		}
	}

	return InResponse{
		Version:  versionFromTag(foundTag),
		Metadata: metadataFromTag(foundTag),
	}, nil
}

// This resource stores the attachments as line-separated markdown links.
func (c *InCommand) getAttachments(releaseBody string) ([]attachment, error) {
	var attachments []attachment

	lines := strings.Split(releaseBody, "\n")
	for _, line := range lines {
		nameStart := strings.Index(line, "[")
		nameEnd := strings.Index(line, "]")
		urlStart := strings.Index(line, "(")
		urlEnd := strings.Index(line, ")")

		if nameStart == -1 || nameEnd == -1 || urlStart == -1 || urlEnd == -1 {
			continue
		}

		nameStart++
		urlStart++

		attachments = append(attachments, attachment{
			Name: line[nameStart:nameEnd],
			URL:  line[urlStart:urlEnd],
		})

	}

	return attachments, nil
}
