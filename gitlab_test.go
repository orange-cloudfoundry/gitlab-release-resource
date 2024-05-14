package resource_test

import (
	"net/http"

	. "github.com/orange-cloudfoundry/gitlab-release-resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	"github.com/xanzy/go-gitlab"
)

var _ = Describe("GitLab Client", func() {
	var server *ghttp.Server
	var client *GitlabClient
	var source Source

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	JustBeforeEach(func() {
		source.GitLabAPIURL = server.URL()
		var err error
		client, err = NewGitLabClient(source)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Context("with bad URLs", func() {
		BeforeEach(func() {
			source.AccessToken = "hello?"
		})
		It("returns an error if the API URL is bad", func() {
			source.GitLabAPIURL = ":"
			_, err := NewGitLabClient(source)
			Ω(err).Should(HaveOccurred())
		})
	})

	Context("with an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				Repository:  "concourse",
				AccessToken: "abc123",
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v4/projects/concourse/repository/tags"),
					ghttp.RespondWith(200, "[]"),
					ghttp.VerifyHeaderKV("Private-Token", "abc123"),
				),
			)
		})

		It("sends one", func() {
			_, err := client.ListTags()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("without an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v4/projects/concourse/repository/tags"),
					ghttp.RespondWith(200, "[]"),
					ghttp.VerifyHeader(http.Header{"Authorization": nil}),
				),
			)
		})

		It("sends one", func() {
			_, err := client.ListTags()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("GetTag", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}
		})

		Context("When GitLab responds successfully", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v4/projects/concourse/repository/tags/some-tag"),
						ghttp.RespondWith(200, `{ "Name": "some-tag" }`),
					),
				)
			})

			It("Returns a populated github.Tag", func() {
				expectedTag := &gitlab.Tag{
					Name: *gitlab.Ptr("some-tag"),
				}
				tag, err := client.GetTag("some-tag")
				Ω(err).ShouldNot(HaveOccurred())
				Expect(tag).To(Equal(expectedTag))
			})
		})

		Context("The tag does not exist", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v4/projects/concourse/repository/tags/some-tag"),
						ghttp.RespondWith(404, `{ "message": "404 Tag Not Found" }`),
					),
				)
			})
			It("Returns the NotFound error", func() {
				_, err := client.GetTag("some-tag")
				Expect(err).To(Equal(NotFound))
			})
		})
	})

	Describe("GetRelease", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}
		})

		Context("When GitLab responds successfully", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v4/projects/concourse/releases/some-tag"),
						ghttp.RespondWith(200, `{ "tag_name": "some-tag" }`),
					),
				)
			})

			It("Returns a populated github.Release", func() {
				expectedRelease := &gitlab.Release{
					TagName: "some-tag",
				}
				release, err := client.GetRelease("some-tag")
				Ω(err).ShouldNot(HaveOccurred())
				Expect(release).To(Equal(expectedRelease))
			})
		})
		Context("When GitLab responds forbidden", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v4/projects/concourse/releases/some-tag"),
						ghttp.RespondWith(403, ``),
					),
				)
			})

			It("Returns an error", func() {
				_, err := client.GetRelease("some-tag")
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
