package resource_test

import (
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/xanzy/go-gitlab"

	"github.com/orange-cloudfoundry/gitlab-release-resource"
	"github.com/orange-cloudfoundry/gitlab-release-resource/fakes"
)

func file(path, contents string) {
	Ω(os.WriteFile(path, []byte(contents), 0644)).Should(Succeed())
}

var _ = Describe("Out Command", func() {
	var (
		command      *resource.OutCommand
		gitlabClient *fakes.FakeGitLab
		sourcesDir   string
		request      resource.OutRequest
	)

	BeforeEach(func() {
		var err error

		gitlabClient = &fakes.FakeGitLab{}
		command = resource.NewOutCommand(gitlabClient, io.Discard)

		sourcesDir, err = os.MkdirTemp("", "gitlab-release")
		Ω(err).ShouldNot(HaveOccurred())

		gitlabClient.CreateReleaseStub = func(name string, tag string, body *string) (*gitlab.Release, error) {
			createdRel := gitlab.Release{}
			createdRel.Name = name
			if body != nil {
				createdRel.Description = *body
			}
			createdRel.TagName = tag
			createdRel.Commit.ID = "a2f4a3"
			return &createdRel, nil
		}

		gitlabClient.CreateTagStub = func(name string, ref string) (*gitlab.Tag, error) {
			return &gitlab.Tag{
				Commit: &gitlab.Commit{
					ID:      ref,
					ShortID: ref,
				},
			}, nil
		}

		gitlabClient.UpdateReleaseStub = func(name string, tag string, body *string) (*gitlab.Release, error) {
			return gitlabClient.CreateReleaseStub(name, tag, body)
		}

		gitlabClient.UploadProjectFileStub = func(file string) (*gitlab.ProjectFile, error) {
			return &gitlab.ProjectFile{
				URL: "/base/" + filepath.Base(file),
			}, nil
		}

		gitlabClient.GetReleaseLinksStub = func(tag string) ([]*gitlab.ReleaseLink, error) {
			return []*gitlab.ReleaseLink{}, nil
		}

		gitlabClient.CreateReleaseLinkStub = func(tag string, name string, url string) (*gitlab.ReleaseLink, error) {
			return &gitlab.ReleaseLink{
				URL:  url,
				Name: name,
			}, nil
		}

		globMatching := filepath.Join(sourcesDir, "great-file.tgz")
		globNotMatching := filepath.Join(sourcesDir, "bad-file.txt")
		file(globMatching, "matching")
		file(globNotMatching, "not matching")
	})

	AfterEach(func() {
		Ω(os.RemoveAll(sourcesDir)).Should(Succeed())
	})

	Context("when the release has already been created", func() {
		assetsLinks1 := []*gitlab.ReleaseLink{
			{
				ID:   456789,
				Name: "unicorns.txt",
			},
			{
				ID:   3450798,
				Name: "rainbows.txt",
			},
		}

		assetsLinks2 := []*gitlab.ReleaseLink{
			{
				ID:   23,
				Name: "rainbow.txt",
			},
		}

		existingReleases := []*gitlab.Release{
			{
				TagName:     "v0.3.12",
				Name:        "v0.3.12-name",
				Description: "basic body1",
			},
			{
				TagName:     "tag2",
				Name:        "name2",
				Description: "basic body2",
			},
		}

		// damn anonymous structs...
		existingReleases[0].Assets.Links = assetsLinks1
		existingReleases[1].Assets.Links = assetsLinks2

		BeforeEach(func() {
			gitlabClient.ListReleasesStub = func() ([]*gitlab.Release, error) {
				return existingReleases, nil
			}

			gitlabClient.GetReleaseStub = func(tag string) (*gitlab.Release, error) {
				for _, r := range existingReleases {
					if r.TagName == tag {
						return r, nil
					}
				}
				return nil, resource.NotFound
			}

			gitlabClient.GetReleaseLinksStub = func(tag string) ([]*gitlab.ReleaseLink, error) {
				for _, r := range existingReleases {
					if tag == r.TagName {
						return r.Assets.Links, nil
					}
				}
				return nil, resource.NotFound
			}

			namePath := filepath.Join(sourcesDir, "name")
			bodyPath := filepath.Join(sourcesDir, "body")
			tagPath := filepath.Join(sourcesDir, "tag")

			file(tagPath, "v0.3.12")
			file(namePath, "v0.3.12-newname")
			file(bodyPath, "this is a great release")

			request = resource.OutRequest{
				Params: resource.OutParams{
					NamePath: "name",
					BodyPath: "body",
					TagPath:  "tag",
				},
			}
		})

		It("deletes the existing assets", func() {
			_, err := command.Run(sourcesDir, request)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(gitlabClient.GetReleaseLinksCallCount()).Should(Equal(1))
			Ω(gitlabClient.GetReleaseLinksArgsForCall(0)).Should(Equal(existingReleases[0].TagName))
			Ω(gitlabClient.DeleteReleaseLinkCallCount()).Should(Equal(2))
			arg1, arg2 := gitlabClient.DeleteReleaseLinkArgsForCall(0)
			Ω(arg1).Should(Equal(existingReleases[0].TagName))
			Ω(arg2.ID).Should(Equal(assetsLinks1[0].ID))
		})

		Context("when a body is not supplied", func() {
			BeforeEach(func() {
				request.Params.BodyPath = ""
			})
			It("does not blow away the body", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(gitlabClient.UpdateReleaseCallCount()).Should(Equal(1))
				name, tag, body := gitlabClient.UpdateReleaseArgsForCall(0)
				Ω(name).Should(Equal("v0.3.12-newname"))
				Ω(tag).Should(Equal("v0.3.12"))
				Ω(body).Should(BeNil())
			})
		})

		Context("when a commitish is not supplied", func() {
			It("updates the existing release", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(gitlabClient.UpdateReleaseCallCount()).Should(Equal(1))
				name, tag, body := gitlabClient.UpdateReleaseArgsForCall(0)
				Ω(tag).Should(Equal("v0.3.12"))
				Ω(name).Should(Equal("v0.3.12-newname"))
				Ω(*body).Should(Equal("this is a great release"))
			})
		})

		Context("when a commitish is supplied", func() {
			BeforeEach(func() {
				commitishPath := filepath.Join(sourcesDir, "commitish")
				file(commitishPath, "1z22f1")
				request.Params.CommitishPath = "commitish"
			})
			It("does not updates the existing release", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(gitlabClient.CreateTagCallCount()).Should(Equal(0))
			})
		})
	})

	Context("when the release has not already been created", func() {
		BeforeEach(func() {
			gitlabClient.GetReleaseStub = func(tag string) (*gitlab.Release, error) {
				return nil, resource.NotFound
			}

			namePath := filepath.Join(sourcesDir, "name")
			tagPath := filepath.Join(sourcesDir, "tag")
			bodyPath := filepath.Join(sourcesDir, "body")
			file(namePath, "v0.3.13")
			file(tagPath, "v0.3.13")
			file(bodyPath, "*markdown*")
			request = resource.OutRequest{
				Params: resource.OutParams{
					NamePath: "name",
					TagPath:  "tag",
					BodyPath: "body",
				},
			}
		})

		Context("when the underlying tag has not already been created", func() {
			BeforeEach(func() {
				gitlabClient.GetTagStub = func(name string) (*gitlab.Tag, error) {
					return nil, resource.NotFound
				}
			})

			Context("with a commitish", func() {
				BeforeEach(func() {
					commitishPath := filepath.Join(sourcesDir, "commitish")
					file(commitishPath, "a2f4a3")
					request.Params.CommitishPath = "commitish"
				})

				It("creates a release on gitlab with the tag on the commitish", func() {
					_, err := command.Run(sourcesDir, request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(gitlabClient.CreateTagCallCount()).Should(Equal(1))
					tagName, ref := gitlabClient.CreateTagArgsForCall(0)
					Ω(tagName).Should(Equal(tagName))
					Ω(ref).Should(Equal("a2f4a3"))

					Ω(gitlabClient.CreateReleaseCallCount()).Should(Equal(1))
					name, tag, body := gitlabClient.CreateReleaseArgsForCall(0)
					Ω(name).Should(Equal("v0.3.13"))
					Ω(tag).Should(Equal("v0.3.13"))
					Ω(*body).Should(Equal("*markdown*"))
				})

				It("has some sweet metadata", func() {
					outResponse, err := command.Run(sourcesDir, request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(outResponse.Metadata).Should(ConsistOf(
						resource.MetadataPair{Name: "tag", Value: "v0.3.13"},
						resource.MetadataPair{Name: "name", Value: "v0.3.13"},
						resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
						resource.MetadataPair{Name: "commit_sha", Value: "a2f4a3"},
					))
				})

			})

			Context("without a commitish", func() {
				It("fails to create the release and the tag", func() {
					_, err := command.Run(sourcesDir, request)
					Ω(err).Should(HaveOccurred())
					Ω(gitlabClient.CreateTagCallCount()).Should(Equal(0))
					Ω(gitlabClient.CreateReleaseCallCount()).Should(Equal(0))
				})
			})
		})

		Context("when the underlying tag has already been created", func() {
			BeforeEach(func() {
				gitlabClient.GetTagStub = func(name string) (*gitlab.Tag, error) {
					return &gitlab.Tag{
						Name: "v0.3.13",
					}, nil
				}
			})

			It("creates a release on gitlab with existing tag", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(gitlabClient.CreateTagCallCount()).Should(Equal(0))
				name, tag, body := gitlabClient.CreateReleaseArgsForCall(0)
				Ω(name).Should(Equal("v0.3.13"))
				Ω(tag).Should(Equal("v0.3.13"))
				Ω(*body).Should(Equal("*markdown*"))
			})

			Context("when the tag_prefix is set", func() {
				BeforeEach(func() {
					namePath := filepath.Join(sourcesDir, "name")
					tagPath := filepath.Join(sourcesDir, "tag")
					file(namePath, "v0.3.13")
					file(tagPath, "0.3.13")
					request = resource.OutRequest{
						Params: resource.OutParams{
							NamePath:  "name",
							TagPath:   "tag",
							TagPrefix: "v",
						},
					}
				})
				It("appends the TagPrefix onto the TagName", func() {
					_, err := command.Run(sourcesDir, request)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(gitlabClient.CreateReleaseCallCount()).Should(Equal(1))
					name, tag, _ := gitlabClient.CreateReleaseArgsForCall(0)
					Ω(name).Should(Equal("v0.3.13"))
					Ω(tag).Should(Equal("v0.3.13"))
				})
			})
		})

		Context("with globs", func() {
			BeforeEach(func() {
				request = resource.OutRequest{
					Params: resource.OutParams{
						NamePath: "name",
						BodyPath: "body",
						TagPath:  "tag",
						Globs: []string{
							"*.tgz",
						},
					},
				}
			})

			It("uploads matching file globs", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(gitlabClient.UploadProjectFileCallCount()).Should(Equal(1))
				file := gitlabClient.UploadProjectFileArgsForCall(0)
				Ω(file).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))
			})

			It("returns an error if a glob is provided that does not match any files", func() {
				request.Params.Globs = []string{
					"*.tgz",
					"*.gif",
				}
				_, err := command.Run(sourcesDir, request)
				Ω(err).Should(HaveOccurred())
				Ω(err).Should(MatchError("could not find file that matches glob '*.gif'"))
			})
		})
	})

})
