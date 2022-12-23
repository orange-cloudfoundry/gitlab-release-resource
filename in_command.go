package resource

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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


func (c *InCommand) matchAsset(name string, globs []string) bool {
	if len(globs) == 0 {
		return true
	}
	for _, glob := range globs {
		matches, _ := filepath.Match(glob, name)
		if matches {
			return true
		}
	}
	return false
}

func (c *InCommand) matchFormat(format string, formats []string) bool {
	for _, f := range formats {
		if f == format {
			return true
		}
	}
	return false
}

func (c *InCommand) Run(destDir string, request InRequest) (InResponse, error) {
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return InResponse{}, err
	}

	if request.Version == nil {
		return InResponse{}, errors.New("missing required Version")
	}

	if request.Version.Tag == "" {
		return InResponse{}, errors.New("missing required Version Tag")
	}

	release, err := c.gitlab.GetRelease(request.Version.Tag)
	if err != nil {
		if errors.Is(err, NotFound) {
			return InResponse{}, errors.New("no releases")
		}
		return InResponse{}, err
	}

	tagPath := filepath.Join(destDir, "tag")
	err = ioutil.WriteFile(tagPath, []byte(release.TagName), 0644)
	if err != nil {
		return InResponse{}, err
	}

	versionParser, err := newVersionParser(request.Source.TagFilter)
	if err != nil {
		return InResponse{}, err
	}
	version := versionParser.parse(release.TagName)
	versionPath := filepath.Join(destDir, "version")
	err = ioutil.WriteFile(versionPath, []byte(version), 0644)
	if err != nil {
		return InResponse{}, err
	}

	commitPath := filepath.Join(destDir, "commit_sha")
	err = ioutil.WriteFile(commitPath, []byte(release.Commit.ID), 0644)
	if err != nil {
		return InResponse{}, err
	}

	body := release.Description
	bodyPath := filepath.Join(destDir, "body")
	err = ioutil.WriteFile(bodyPath, []byte(body), 0644)
	if err != nil {
		return InResponse{}, err
	}

	for _, asset := range release.Assets.Links {
		path := filepath.Join(destDir, asset.Name)
		if !c.matchAsset(asset.Name, request.Params.Globs) {
			continue
		}

		err := c.gitlab.DownloadProjectFile(asset.URL, path)
		if err != nil {
			return InResponse{}, err
		}
	}

	sources := request.Params.IncludeSources
	if len(sources) == 0 {
		if request.Params.IncludeSourceTarball {
			sources = append(sources, "tar.gz")
		}
		if request.Params.IncludeSourceZip {
			sources = append(sources, "zip")
		}
	}

	for _, source := range release.Assets.Sources {
		if !c.matchFormat(source.Format, sources) {
			continue
		}

		name := path.Base(source.URL)
		path := filepath.Join(destDir, name)
		err := c.gitlab.DownloadProjectFile(source.URL, path)
		if err != nil {
			return InResponse{}, err
		}
	}

	return InResponse{
		Version:  versionFromRelease(release),
		Metadata: metadataFromRelease(release),
	}, nil
}
