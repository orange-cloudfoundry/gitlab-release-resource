package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/xanzy/go-gitlab"
	"github.com/orange-cloudfoundry/gitlab-release-resource"
	"github.com/orange-cloudfoundry/gitlab-release-resource/fakes"
)

func v2r(versions []string) []*gitlab.Release {
	res := []*gitlab.Release{}
	for _, version := range versions {
		res = append(res, &gitlab.Release{
			Name: version,
			TagName: version,
			Commit: gitlab.Commit{
				ID: "dabdab",
			},
		})
	}
	return res
}

var _ = Describe("Check Command", func() {
	var (
		gitlabClient *fakes.FakeGitLab
		command      *resource.CheckCommand
		request      *resource.CheckRequest
	)

	no_version := []string{}
	one_version := []string{"v1.0.0"}
	many_version := []string{
		"v1.0.0",
		"v2.1.10",
		"v5.1.0",
		"v1.0.1",
		"v5.0.0",
		"v1.1.0",
		"v2.5.1",
	}

	BeforeEach(func() {
		gitlabClient = &fakes.FakeGitLab{}
		command = resource.NewCheckCommand(gitlabClient)
		request = &resource.CheckRequest{}
	})

	Context("When no version are available", func() {
		BeforeEach(func() {
			gitlabClient.ListReleasesReturns(v2r(no_version), nil)
		})

		Context("when this is the first time that the resource has been run", func() {
			BeforeEach(func() {
				request.Version = resource.Version{}
			})

			It("detects no version", func() {
				versions, err := command.Run(*request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(BeEmpty())
			})
		})

		Context("when the resource has already been run", func() {
			BeforeEach(func() {
				request.Version = resource.Version{
					Tag: "v0.0.0",
					CommitSHA: "dabdab",
				}
			})

			It("detects no version", func() {
				versions, err := command.Run(*request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(BeEmpty())
			})
		})
	})

	Context("When one version is available", func() {
		BeforeEach(func() {
			gitlabClient.ListReleasesReturns(v2r(one_version), nil)
		})

		Context("when this is the first time that the resource has been run", func() {
			BeforeEach(func() {
				request.Version = resource.Version{}
			})
			It("detects a single version", func() {
				versions, err := command.Run(*request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(HaveLen(1))
				Ω(versions).Should(Equal([]resource.Version{
					resource.Version{Tag: "v1.0.0", CommitSHA: "dabdab"},
				}))
			})
		})

		Context("when the resource has already been run", func() {
			BeforeEach(func() {
				request.Version = resource.Version{
					Tag: "v0.0.0",
					CommitSHA: "dabdab",
				}
			})
			It("detects a single version", func() {
				versions, err := command.Run(*request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(HaveLen(1))
				Ω(versions).Should(Equal([]resource.Version{
					{Tag: "v1.0.0", CommitSHA: "dabdab"},
				}))
			})
		})

		Context("when the resource has already been run with last version", func() {
			BeforeEach(func() {
				request.Version = resource.Version{
					Tag: "v1.0.0",
					CommitSHA: "dabdab",
				}
			})
			It("detects the requested version", func() {
				versions, err := command.Run(*request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(HaveLen(1))
				Ω(versions).Should(Equal([]resource.Version{
					{Tag: "v1.0.0", CommitSHA: "dabdab"},
				}))
			})
		})
	})

	Context("When many version are available", func() {
		BeforeEach(func() {
			gitlabClient.ListReleasesReturns(v2r(many_version), nil)
		})

		Context("when this is the first time that the resource has been run", func() {
			BeforeEach(func() {
				request.Version = resource.Version{}
			})
			It("detects a last available version", func() {
				versions, err := command.Run(*request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(HaveLen(1))
				Ω(versions).Should(Equal([]resource.Version{
					{Tag: "v5.1.0", CommitSHA: "dabdab"},
				}))
			})
		})

		Context("when the resource has already been run", func() {

			Context("when all versions are new", func() {
				BeforeEach(func() {
					request.Version.Tag = "v0.0.1"
				})
				It("detects all versions correclty", func() {
					versions, err := command.Run(*request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(versions).Should(HaveLen(7))
					Ω(versions).Should(Equal([]resource.Version{
						{Tag: "v1.0.0",  CommitSHA: "dabdab"},
						{Tag: "v1.0.1",  CommitSHA: "dabdab"},
						{Tag: "v1.1.0",  CommitSHA: "dabdab"},
						{Tag: "v2.1.10", CommitSHA: "dabdab"},
						{Tag: "v2.5.1",  CommitSHA: "dabdab"},
						{Tag: "v5.0.0",  CommitSHA: "dabdab"},
						{Tag: "v5.1.0",  CommitSHA: "dabdab"},
					}))
				})
			})

			Context("when requested version has already been detected", func() {
				BeforeEach(func() {
					request.Version.Tag = "v5.1.0"
				})
				It("replies with already known version", func() {
					versions, err := command.Run(*request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(versions).Should(HaveLen(1))
					Ω(versions).Should(Equal([]resource.Version{
						{Tag: "v5.1.0",  CommitSHA: "dabdab"},
					}))
				})
			})

			Context("when requested version has been deleted", func() {
				BeforeEach(func() {
					request.Version.Tag = "v2.5.0"
				})
				It("replies with all version greater than requested", func() {
					versions, err := command.Run(*request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(versions).Should(HaveLen(3))
					Ω(versions).Should(Equal([]resource.Version{
						{Tag: "v2.5.1",  CommitSHA: "dabdab"},
						{Tag: "v5.0.0",  CommitSHA: "dabdab"},
						{Tag: "v5.1.0",  CommitSHA: "dabdab"},
					}))
				})
			})

			Context("when providing a custom tag filter", func() {
				BeforeEach(func() {
					request.Source.TagFilter = `^v([0-9]+\.1\.[0-9]+)`
					request.Version.Tag = "v0.1.0"
				})
				It("replies with all version greater than requested", func() {
					versions, err := command.Run(*request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(versions).Should(HaveLen(3))
					Ω(versions).Should(Equal([]resource.Version{
						{Tag: "v1.1.0",   CommitSHA: "dabdab"},
						{Tag: "v2.1.10",  CommitSHA: "dabdab"},
						{Tag: "v5.1.0",   CommitSHA: "dabdab"},
					}))
				})
			})
		})
	})


	Context("When dealing with complex postrelease and prerelease versions", func() {
		BeforeEach(func() {
			messy_version := append(many_version, []string{
				"v1.0.0-dev1",
				"v1.0.0_ora-dev1",
				"production",
				"v1.0.0_ora-dev2",
				"v1.0.0_ora",
				"v1.0.0-dev2",
			}...)

			gitlabClient.ListReleasesReturns(v2r(messy_version), nil)
		})


		Context("when the resource has already been run", func() {

			Context("when all versions are new", func() {
				BeforeEach(func() {
					request.Version.Tag = "v0.0.1"
					request.Source.TagFilter = "v?([0-9]+.*)"
				})
				It("detects all versions correclty", func() {
					versions, err := command.Run(*request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(versions).Should(Equal([]resource.Version{
						{Tag: "v1.0.0-dev1",  CommitSHA: "dabdab"},
						{Tag: "v1.0.0-dev2",  CommitSHA: "dabdab"},
						{Tag: "v1.0.0",  CommitSHA: "dabdab"},
						{Tag: "v1.0.1",  CommitSHA: "dabdab"},
						{Tag: "v1.0.0_ora-dev1",  CommitSHA: "dabdab"},
						{Tag: "v1.0.0_ora-dev2",  CommitSHA: "dabdab"},
						{Tag: "v1.0.0_ora",  CommitSHA: "dabdab"},
						{Tag: "v1.1.0",  CommitSHA: "dabdab"},
						{Tag: "v2.1.10", CommitSHA: "dabdab"},
						{Tag: "v2.5.1",  CommitSHA: "dabdab"},
						{Tag: "v5.0.0",  CommitSHA: "dabdab"},
						{Tag: "v5.1.0",  CommitSHA: "dabdab"},
					}))
				})
			})
		})
	})

})
