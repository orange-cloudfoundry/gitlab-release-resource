package resource_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	. "github.com/orange-cloudfoundry/gitlab-release-resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var _ = Describe("GitLab Client", func() {
	var server *ghttp.Server
	var client *GitlabClient
	var source Source

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	JustBeforeEach(func() {
		if source.GitLabAPIURL == "" {
			source.GitLabAPIURL = server.URL()
		}
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
			It("Returns the ErrNotFound error", func() {
				_, err := client.GetTag("some-tag")
				Expect(err).To(Equal(ErrNotFound))
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

	Describe("DownloadProjectFile", func() {
		var (
			tmpDir   string
			destPath string
		)

		BeforeEach(func() {
			source = Source{
				Repository:  "concourse",
				AccessToken: "abc123",
			}

			var err error
			tmpDir, err = os.MkdirTemp("", "gitlab-download")
			Ω(err).ShouldNot(HaveOccurred())
			destPath = filepath.Join(tmpDir, "asset.bin")
		})

		AfterEach(func() {
			Ω(os.RemoveAll(tmpDir)).Should(Succeed())
		})

		Context("when downloading from GitLab host", func() {
			It("uses Private-Token and no basic auth", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/uploads/hash/asset.bin"),
						ghttp.VerifyHeaderKV("Private-Token", "abc123"),
						ghttp.VerifyHeader(http.Header{"Authorization": nil}),
						ghttp.RespondWith(200, "downloaded-from-gitlab"),
					),
				)

				err := client.DownloadProjectFile(server.URL()+"/api/v4/uploads/hash/asset.bin", destPath)
				Ω(err).ShouldNot(HaveOccurred())

				contents, err := os.ReadFile(destPath)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("downloaded-from-gitlab"))
			})
		})

		Context("when downloading from external host with configured auth", func() {
			var externalServer *ghttp.Server

			BeforeEach(func() {
				externalServer = ghttp.NewServer()
				externalURL, err := url.Parse(externalServer.URL())
				Ω(err).ShouldNot(HaveOccurred())
				source.GitLabAPIURL = "https://gitlab.example.internal"
				source.DownloadAuths = []DownloadAuth{{
					Host:     externalURL.Hostname(),
					Username: "ext-user",
					Password: "ext-pass",
				}}

				client, err = NewGitLabClient(source)
				Ω(err).ShouldNot(HaveOccurred())
			})

			AfterEach(func() {
				externalServer.Close()
			})

			It("uses basic auth and does not leak Private-Token", func() {
				externalServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/files/asset.bin"),
						ghttp.VerifyBasicAuth("ext-user", "ext-pass"),
						ghttp.VerifyHeader(http.Header{"Private-Token": nil}),
						ghttp.RespondWith(200, "downloaded-from-external"),
					),
				)



				err := client.DownloadProjectFile(externalServer.URL()+"/files/asset.bin", destPath)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when downloading from external host without configured auth", func() {
			var externalServer *ghttp.Server

			BeforeEach(func() {
				externalServer = ghttp.NewServer()
				source.GitLabAPIURL = "https://gitlab.example.internal"

				var err error
				client, err = NewGitLabClient(source)
				Ω(err).ShouldNot(HaveOccurred())
			})

			AfterEach(func() {
				externalServer.Close()
			})

			It("sends no authentication header", func() {
				externalServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/files/public.bin"),
						ghttp.VerifyHeader(http.Header{"Authorization": nil}),
						ghttp.VerifyHeader(http.Header{"Private-Token": nil}),
						ghttp.RespondWith(200, "public-file"),
					),
				)

				err := client.DownloadProjectFile(externalServer.URL()+"/files/public.bin", destPath)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when download returns a non-200 status", func() {
			for _, tc := range []struct {
				status int
				label  string
			}{
				{status: http.StatusUnauthorized, label: "401"},
				{status: http.StatusNotFound, label: "404"},
				{status: http.StatusInternalServerError, label: "500"},
			} {
				tc := tc
				It(fmt.Sprintf("returns an error for HTTP %s", tc.label), func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/uploads/hash/asset.bin"),
							ghttp.VerifyHeaderKV("Private-Token", "abc123"),
							ghttp.RespondWith(tc.status, ""),
						),
					)

					err := client.DownloadProjectFile(server.URL()+"/api/v4/uploads/hash/asset.bin", destPath)
					Ω(err).Should(MatchError(fmt.Sprintf("failed to download file `asset.bin`: HTTP status %d", tc.status)))
				})
			}
		})
	})
})
