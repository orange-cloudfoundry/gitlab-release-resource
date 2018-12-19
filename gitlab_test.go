package resource_test

import (
	"net/http"

	. "github.com/edtan/gitlab-release-resource"

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
					ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases"),
					ghttp.RespondWith(200, "[]"),
					ghttp.VerifyHeaderKV("Authorization", "Bearer abc123"),
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
					ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases"),
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

	Describe("GetRelease", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}
		})
	})

	Describe("GetReleaseByTag", func() {
		BeforeEach(func() {
			source = Source{
				Repository: "concourse",
			}
		})

		Context("When GitLab responds successfully", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases/tags/some-tag"),
						ghttp.RespondWith(200, `{ "id": "1" }`),
					),
				)
			})

			It("Returns a populated github.Tag", func() {
				expectedRelease := &gitlab.Tag{
					Name: *gitlab.String("1"),
				}

				release, err := client.GetTag("some-tag")

				Ω(err).ShouldNot(HaveOccurred())
				Expect(release).To(Equal(expectedRelease))
			})
		})
	})
})
