package resource_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/xanzy/go-gitlab"
)

func TestGithubReleaseResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GitLab Release Resource Suite")
}

func newTag(name, sha string) *gitlab.Tag {
	return &gitlab.Tag{
		Commit: &gitlab.Commit{
			ID: *gitlab.String(sha),
		},
		Name: *gitlab.String(name),
	}
}
