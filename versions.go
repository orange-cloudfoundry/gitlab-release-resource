package resource

import (
	"regexp"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var defaultTagFilter = "^v?([^v].*)"

type versionParser struct {
	re *regexp.Regexp
}

func newVersionParser(filter string) (versionParser, error) {
	if filter == "" {
		filter = defaultTagFilter
	}
	re, err := regexp.Compile(filter)
	if err != nil {
		return versionParser{}, err
	}
	return versionParser{re: re}, nil
}

func (vp *versionParser) parse(tag string) string {
	matches := vp.re.FindStringSubmatch(tag)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	return ""
}

func versionFromRelease(release *gitlab.Release) Version {
	return Version{
		Tag:       release.TagName,
		CommitSHA: release.Commit.ID,
	}
}
