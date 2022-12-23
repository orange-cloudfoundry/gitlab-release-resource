package resource

import "github.com/xanzy/go-gitlab"

func metadataFromRelease(release *gitlab.Release) []MetadataPair {
	metadata := []MetadataPair{
		{
			Name:  "name",
			Value: release.Name,
		},
		{
			Name:  "tag",
			Value: release.TagName,
		},
	}

	if release.Description != "" {
		metadata = append(metadata, MetadataPair{
			Name:     "body",
			Value:    release.Description,
			Markdown: true,
		})
	}
	if release.Commit.ID != "" {
		metadata = append(metadata, MetadataPair{
			Name:  "commit_sha",
			Value: release.Commit.ID,
		})
	}
	return metadata
}
