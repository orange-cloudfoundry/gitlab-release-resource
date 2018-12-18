package resource

import (
	"regexp"

	"github.com/xanzy/go-gitlab"
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

func versionFromTag(tag *gitlab.Tag) Version {
	return Version{
		Tag:       tag.Name,
		CommitSHA: tag.Commit.ID,
	}
}
