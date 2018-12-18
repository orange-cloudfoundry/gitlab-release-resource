package resource

import (
	"sort"

	"github.com/cppforlife/go-semi-semantic/version"
	"github.com/xanzy/go-gitlab"
)

type CheckCommand struct {
	gitlab GitLab
}

func NewCheckCommand(gitlab GitLab) *CheckCommand {
	return &CheckCommand{
		gitlab: gitlab,
	}
}

func (c *CheckCommand) Run(request CheckRequest) ([]Version, error) {
	var tags []*gitlab.Tag
	var err error
	if (request.Version == Version{}) {
		tags, err = c.gitlab.ListTags()
	} else {
		tags, err = c.gitlab.ListTagsUntil(request.Version.Tag)
	}

	if err != nil {
		return []Version{}, err
	}

	if len(tags) == 0 {
		return []Version{}, nil
	}

	var filteredTags []*gitlab.Tag

	// TODO: make ListTagsUntil work better with this
	versionParser, err := newVersionParser(request.Source.TagFilter)
	if err != nil {
		return []Version{}, err
	}

	for _, tag := range tags {
		if _, err := version.NewVersionFromString(versionParser.parse(tag.Name)); err != nil {
			continue
		}

		if tag.Release == nil {
			continue
		}

		filteredTags = append(filteredTags, tag)
	}

	sort.Slice(filteredTags, func(i, j int) bool {
		first, err := version.NewVersionFromString(versionParser.parse(filteredTags[i].Name))
		if err != nil {
			return true
		}

		second, err := version.NewVersionFromString(versionParser.parse(filteredTags[j].Name))
		if err != nil {
			return false
		}

		return first.IsLt(second)
	})

	if len(filteredTags) == 0 {
		return []Version{}, nil
	}
	latestTag := filteredTags[len(filteredTags)-1]

	if (request.Version == Version{}) {
		return []Version{versionFromTag(latestTag)}, nil
	}

	if latestTag.Name == request.Version.Tag {
		// GitHub release resource returns empty array:
		// https://github.com/concourse/github-release-resource/blob/master/check_command.go#L87
		// but documentation says to return current item?
		// https://concourse-ci.org/implementing-resources.html#section_resource-check
		return []Version{versionFromTag(latestTag)}, nil
	}

	upToLatest := false
	nextVersions := []Version{} // contains the requested version and all newer ones

	for _, release := range filteredTags {
		if !upToLatest {
			version := release.Name
			upToLatest = request.Version.Tag == version
		}

		if upToLatest {
			nextVersions = append(nextVersions, Version{Tag: release.Name})
		}
	}

	if !upToLatest {
		// current version was removed; start over from latest
		nextVersions = append(
			nextVersions,
			versionFromTag(filteredTags[len(filteredTags)-1]),
		)
	}

	return nextVersions, nil
}
