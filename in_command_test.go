package resource_test

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	resource "github.com/orange-cloudfoundry/gitlab-release-resource"
	"github.com/orange-cloudfoundry/gitlab-release-resource/fakes"
)

var _ = Describe("In Command", func() {
	var (
		command      *resource.InCommand
		gitlabClient *fakes.FakeGitLab
		inRequest    resource.InRequest
		inResponse   resource.InResponse
		inErr        error
		tmpDir       string
		destDir      string
	)

	BeforeEach(func() {
		var err error
		gitlabClient = &fakes.FakeGitLab{}
		command = resource.NewInCommand(gitlabClient, io.Discard)
		inErr = nil
		tmpDir, err = os.MkdirTemp("", "gitlab-release")
		Ω(err).ShouldNot(HaveOccurred())
		destDir = filepath.Join(tmpDir, "destination")
		gitlabClient.DownloadProjectFileReturns(nil)
		inRequest = resource.InRequest{}
		inResponse = resource.InResponse{}
	})

	AfterEach(func() {
		Ω(os.RemoveAll(tmpDir)).Should(Succeed())
	})

	buildRelease := func(tag, sha string) *gitlab.Release {
		r := &gitlab.Release{
			TagName:     tag,
			Name:        tag,
			Description: "*markdown*",
			Commit: gitlab.Commit{
				ID: sha,
			},
		}
		r.Assets.Links = []*gitlab.ReleaseLink{
			{ID: 1, Name: "example.txt", URL: "example.txt"},
			{ID: 2, Name: "example.rtf", URL: "example.rtf"},
			{ID: 3, Name: "example.png", URL: "example.png"},
		}
		data := []byte(`
    [
      { "format": "zip",    "url": "sources.zip" },
      { "format": "tar.gz", "url": "sources.tar.gz" },
      { "format": "tar.bz2","url": "sources.tar.bz2" },
      { "format": "tar",    "url": "sources.tar" }
    ]
`)
		err := json.Unmarshal(data, &r.Assets.Sources)
		Ω(err).ShouldNot(HaveOccurred())
		return r
	}

	Context("when a tagged release is found", func() {
		BeforeEach(func() {
			gitlabClient.GetReleaseReturns(buildRelease("v0.35.0", "abc123"), nil)
			inRequest.Version = &resource.Version{
				Tag: "v0.35.0",
			}
		})

		Context("when valid asset filename globs are given", func() {
			BeforeEach(func() {
				inRequest.Params = resource.InParams{
					Globs: []string{"*.txt", "*.rtf"},
				}
			})

			It("succeeds", func() {
				inResponse, inErr = command.Run(destDir, inRequest)
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("in answer with version", func() {
				inResponse, inErr = command.Run(destDir, inRequest)
				Ω(inErr).ShouldNot(HaveOccurred())
				Ω(inResponse.Version).Should(Equal(resource.Version{
					Tag:       "v0.35.0",
					CommitSHA: "abc123",
				}))
			})

			It("with sweet metadata", func() {
				inResponse, inErr = command.Run(destDir, inRequest)
				Ω(inErr).ShouldNot(HaveOccurred())
				Ω(inResponse.Metadata).Should(ConsistOf([]resource.MetadataPair{
					{Name: "name", Value: "v0.35.0"},
					{Name: "tag", Value: "v0.35.0"},
					{Name: "commit_sha", Value: "abc123"},
					{Name: "body", Value: "*markdown*", Markdown: true},
				}))
			})

			It("calls #GetRelease with the correct arguments", func() {
				inResponse, inErr = command.Run(destDir, inRequest)
				Ω(inErr).ShouldNot(HaveOccurred())
				Ω(gitlabClient.GetReleaseCallCount()).Should(Equal(1))
				Ω(gitlabClient.GetReleaseArgsForCall(0)).Should(Equal("v0.35.0"))
			})

			It("downloads only the files that match the globs", func() {
				inResponse, inErr = command.Run(destDir, inRequest)
				Ω(inErr).ShouldNot(HaveOccurred())

				Expect(gitlabClient.DownloadProjectFileCallCount()).To(Equal(2))
				arg1, arg2 := gitlabClient.DownloadProjectFileArgsForCall(0)
				Ω(arg1).Should(Equal("example.txt"))
				Ω(arg2).Should(Equal(path.Join(destDir, "example.txt")))
				arg1, arg2 = gitlabClient.DownloadProjectFileArgsForCall(1)
				Ω(arg1).Should(Equal("example.rtf"))
				Ω(arg2).Should(Equal(path.Join(destDir, "example.rtf")))
			})

			It("does create the body, tag and version files", func() {
				inResponse, inErr = command.Run(destDir, inRequest)

				contents, err := os.ReadFile(path.Join(destDir, "tag"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("v0.35.0"))

				contents, err = os.ReadFile(path.Join(destDir, "version"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("0.35.0"))

				contents, err = os.ReadFile(path.Join(destDir, "commit_sha"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("abc123"))

				contents, err = os.ReadFile(path.Join(destDir, "body"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("*markdown*"))
			})

			Context("when there is a custom tag filter", func() {
				BeforeEach(func() {
					inRequest.Source = resource.Source{
						TagFilter: "package-(.*)",
					}
					gitlabClient.GetReleaseReturns(buildRelease("package-0.35.0", "abc123"), nil)
				})

				It("succeeds", func() {
					inResponse, inErr = command.Run(destDir, inRequest)
					Expect(inErr).ToNot(HaveOccurred())
				})

				It("does create the body, tag and version files", func() {
					inResponse, inErr = command.Run(destDir, inRequest)
					contents, err := os.ReadFile(path.Join(destDir, "tag"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("package-0.35.0"))
					contents, err = os.ReadFile(path.Join(destDir, "version"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("0.35.0"))
				})
			})

		})

		Context("when no globs are specified", func() {
			BeforeEach(func() {
				inRequest.Params.Globs = []string{}
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(
					resource.Version{Tag: "v0.35.0", CommitSHA: "abc123"},
				))
			})

			It("downloads all of the files", func() {
				Expect(gitlabClient.DownloadProjectFileCallCount()).To(Equal(3))

				arg1, arg2 := gitlabClient.DownloadProjectFileArgsForCall(0)
				Ω(arg1).Should(Equal("example.txt"))
				Ω(arg2).Should(Equal(path.Join(destDir, "example.txt")))

				arg1, arg2 = gitlabClient.DownloadProjectFileArgsForCall(1)
				Ω(arg1).Should(Equal("example.rtf"))
				Ω(arg2).Should(Equal(path.Join(destDir, "example.rtf")))

				arg1, arg2 = gitlabClient.DownloadProjectFileArgsForCall(2)
				Ω(arg1).Should(Equal("example.png"))
				Ω(arg2).Should(Equal(path.Join(destDir, "example.png")))
			})
		})

		Context("when asking for sources", func() {
			BeforeEach(func() {
				inRequest.Params.Globs = []string{"does-not-match"}
			})

			Context("specifying all possible formats", func() {
				BeforeEach(func() {
					inRequest.Params.IncludeSources = []string{"zip", "tar.bz2", "tar.gz", "tar"}
				})

				It("downloads all the available sources", func() {
					inResponse, inErr = command.Run(destDir, inRequest)
					Ω(inErr).ShouldNot(HaveOccurred())

					Expect(gitlabClient.DownloadProjectFileCallCount()).To(Equal(4))
					arg1, arg2 := gitlabClient.DownloadProjectFileArgsForCall(0)
					Ω(arg1).Should(Equal("sources.zip"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.zip")))
					arg1, arg2 = gitlabClient.DownloadProjectFileArgsForCall(1)
					Ω(arg1).Should(Equal("sources.tar.gz"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.tar.gz")))
					arg1, arg2 = gitlabClient.DownloadProjectFileArgsForCall(2)
					Ω(arg1).Should(Equal("sources.tar.bz2"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.tar.bz2")))
					arg1, arg2 = gitlabClient.DownloadProjectFileArgsForCall(3)
					Ω(arg1).Should(Equal("sources.tar"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.tar")))
				})
			})

			Context("specifying some formats", func() {
				BeforeEach(func() {
					inRequest.Params.IncludeSources = []string{"tar.bz2", "tar.gz"}
				})
				It("downloads only the requested source formats", func() {
					inResponse, inErr = command.Run(destDir, inRequest)
					Ω(inErr).ShouldNot(HaveOccurred())
					Expect(gitlabClient.DownloadProjectFileCallCount()).To(Equal(2))
					arg1, arg2 := gitlabClient.DownloadProjectFileArgsForCall(0)
					Ω(arg1).Should(Equal("sources.tar.gz"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.tar.gz")))
					arg1, arg2 = gitlabClient.DownloadProjectFileArgsForCall(1)
					Ω(arg1).Should(Equal("sources.tar.bz2"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.tar.bz2")))
				})
			})

			Context("using tarball switch", func() {
				BeforeEach(func() {
					inRequest.Params.IncludeSourceTarball = true
				})
				It("downloads only the requested source formats", func() {
					inResponse, inErr = command.Run(destDir, inRequest)
					Ω(inErr).ShouldNot(HaveOccurred())
					Expect(gitlabClient.DownloadProjectFileCallCount()).To(Equal(1))
					arg1, arg2 := gitlabClient.DownloadProjectFileArgsForCall(0)
					Ω(arg1).Should(Equal("sources.tar.gz"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.tar.gz")))
				})
			})

			Context("using zip switch", func() {
				BeforeEach(func() {
					inRequest.Params.IncludeSourceZip = true
				})
				It("downloads only the requested source formats", func() {
					inResponse, inErr = command.Run(destDir, inRequest)
					Ω(inErr).ShouldNot(HaveOccurred())
					Expect(gitlabClient.DownloadProjectFileCallCount()).To(Equal(1))
					arg1, arg2 := gitlabClient.DownloadProjectFileArgsForCall(0)
					Ω(arg1).Should(Equal("sources.zip"))
					Ω(arg2).Should(Equal(path.Join(destDir, "sources.zip")))
				})
			})
		})

		Context("when downloading an asset fails", func() {
			BeforeEach(func() {
				gitlabClient.DownloadProjectFileReturns(errors.New("not this time"))
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("returns an error", func() {
				Ω(inErr).Should(HaveOccurred())
			})
		})
	})

	Context("when no tagged release is present", func() {
		BeforeEach(func() {
			gitlabClient.GetReleaseReturns(nil, resource.ErrNotFound)
			inRequest.Version = &resource.Version{
				Tag: "v0.40.0",
			}
			inResponse, inErr = command.Run(destDir, inRequest)
		})
		It("returns an error", func() {
			Ω(inErr).Should(MatchError("no releases"))
		})
	})

	Context("when getting a tagged release fails", func() {
		disaster := errors.New("nope")
		BeforeEach(func() {
			gitlabClient.GetReleaseReturns(nil, disaster)
			inRequest.Version = &resource.Version{
				Tag: "some-tag",
			}
			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("returns the error", func() {
			Ω(inErr).Should(Equal(disaster))
		})
	})

	Context("with incomplete input JSON", func() {
		Context("is missing version", func() {
			It("complain about it", func() {
				inResponse, inErr = command.Run(destDir, inRequest)
				Ω(inErr).Should(HaveOccurred())
			})
		})
		Context("is missing version tag", func() {
			It("complain about it", func() {
				inRequest.Version = &resource.Version{}
				inResponse, inErr = command.Run(destDir, inRequest)
				Ω(inErr).Should(HaveOccurred())
			})
		})
	})
})
