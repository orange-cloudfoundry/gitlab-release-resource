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
	versionParser, err := newVersionParser(request.Source.TagFilter)

	// fetch available releaes
	releases, err := c.gitlab.ListReleases()
	if err != nil {
		return []Version{}, err
	}

	// filter releases
	filteredReleases := []*gitlab.Release{}
	targetVersion, err := version.NewVersionFromString(versionParser.parse(request.Version.Tag))
	if (request.Version != Version{}) && err != nil {
		return []Version{}, err
	}

	for _, r := range releases {
		current, err := version.NewVersionFromString(versionParser.parse(r.TagName))
		// must match tag regex
		if err != nil {
			continue
		}
		// when given, keep only releases greater-or-equal than target version
		if ((request.Version == Version{}) || !current.IsLt(targetVersion)) {
			filteredReleases = append(filteredReleases, r)
		}
	}

	// sort releases from older to newer
	sort.Slice(filteredReleases, func(i, j int) bool {
		// errors ingored since has already been filtered out by regexp
		first, _ := version.NewVersionFromString(versionParser.parse(filteredReleases[i].Name))
		second, _ := version.NewVersionFromString(versionParser.parse(filteredReleases[j].Name))
		return first.IsLt(second)
	})

	// no version available
	if len(filteredReleases) == 0 {
		return []Version{}, nil
	}

	// first check, no target version given, reply last available release
	latestRelease := filteredReleases[len(filteredReleases) - 1]
	if (request.Version == Version{}) {
		return []Version{versionFromRelease(latestRelease)}, nil
	}

	// built list of next available versions
	nextVersions := []Version{}
	for _, r := range filteredReleases {
		nextVersions = append(nextVersions, versionFromRelease(r))
	}
	return nextVersions, nil
}
