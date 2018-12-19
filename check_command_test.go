package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/xanzy/go-gitlab"

	"github.com/edtan/gitlab-release-resource"
	"github.com/edtan/gitlab-release-resource/fakes"
)

var _ = Describe("Check Command", func() {
	var (
		gitlabClient *fakes.FakeGitLab
		command      *resource.CheckCommand

		returnedTags []*gitlab.Tag
	)

	BeforeEach(func() {
		gitlabClient = &fakes.FakeGitLab{}
		command = resource.NewCheckCommand(gitlabClient)

		returnedTags = []*gitlab.Tag{}
	})

	JustBeforeEach(func() {
		gitlabClient.ListTagsReturns(returnedTags, nil)
	})

	Context("when this is the first time that the resource has been run", func() {
		Context("when there are no releases", func() {
			BeforeEach(func() {
				returnedTags = []*gitlab.Tag{}
			})

			It("returns no versions", func() {
				versions, err := command.Run(resource.CheckRequest{})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(BeEmpty())
			})
		})

		Context("when there are releases", func() {
			BeforeEach(func() {
				returnedTags = []*gitlab.Tag{
					newTag("v0.4.0", "abc123"),
					newTag("0.1.3", "bdc234"),
					newTag("v0.1.2", "cde345"),
				}
			})

			It("outputs the most recent version only", func() {
				command := resource.NewCheckCommand(gitlabClient)

				response, err := command.Run(resource.CheckRequest{})
				Ω(err).ShouldNot(HaveOccurred())

				Ω(response).Should(HaveLen(1))
				Ω(response[0]).Should(Equal(resource.Version{
					Tag: "v0.4.0",
				}))
			})
		})
	})

	Context("when there are prior versions", func() {
		Context("when there are no releases", func() {
			BeforeEach(func() {
				returnedTags = []*gitlab.Tag{}
			})

			It("returns no versions", func() {
				versions, err := command.Run(resource.CheckRequest{})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(BeEmpty())
			})
		})

		Context("when there are releases", func() {
			Context("and there is a custom tag filter", func() {
				BeforeEach(func() {
					returnedTags = []*gitlab.Tag{
						newTag("package-0.1.4", "abc123"),
						newTag("package-0.4.0", "bcd234"),
						newTag("package-0.1.3", "cde345"),
						newTag("package-0.1.2", "def456"),
					}
				})

				It("returns all of the versions that are newer", func() {
					command := resource.NewCheckCommand(gitlabClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "package-0.1.3",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{Tag: "package-0.1.3"},
						{Tag: "package-0.1.4"},
						{Tag: "package-0.4.0"},
					}))
				})
			})

			Context("and the releases do not contain a draft release", func() {
				BeforeEach(func() {
					returnedTags = []*gitlab.Tag{
						newTag("v0.1.4", "abc123"),
						newTag("0.4.0", "bcd234"),
						newTag("v0.1.3", "cde345"),
						newTag("0.1.2", "def456"),
					}
				})

				It("returns an empty list if the latest version has been checked", func() {
					command := resource.NewCheckCommand(gitlabClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "0.4.0",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(BeEmpty())
				})

				It("returns all of the versions that are newer", func() {
					command := resource.NewCheckCommand(gitlabClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "v0.1.3",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{Tag: "v0.1.3"},
						{Tag: "v0.1.4"},
						{Tag: "0.4.0"},
					}))
				})

				It("returns the latest version if the current version is not found", func() {
					command := resource.NewCheckCommand(gitlabClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "v3.4.5",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{Tag: "0.4.0"},
					}))
				})

				Context("when there are not-quite-semver versions", func() {
					BeforeEach(func() {
						returnedTags = append(returnedTags, newTag("v1", "abc123"))
						returnedTags = append(returnedTags, newTag("v0", "bcd234"))
					})

					It("combines them with the semver versions in a reasonable order", func() {
						command := resource.NewCheckCommand(gitlabClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{
								Tag: "v0.1.3",
							},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "v0.1.3"},
							{Tag: "v0.1.4"},
							{Tag: "0.4.0"},
							{Tag: "v1"},
						}))
					})
				})
			})
		})
	})
})
