package resource

import "github.com/xanzy/go-gitlab"

func metadataFromTag(tag *gitlab.Tag) []MetadataPair {
	metadata := []MetadataPair{}

	if tag.Name != "" {
		nameMeta := MetadataPair{
			Name:  "tag",
			Value: tag.Name,
		}

		metadata = append(metadata, nameMeta)
	}

	if tag.Release != nil && tag.Release.Description != "" {
		metadata = append(metadata, MetadataPair{
			Name:     "body",
			Value:    tag.Release.Description,
			Markdown: true,
		})
	}

	if tag.Commit != nil && tag.Commit.ID != "" {
		metadata = append(metadata, MetadataPair{
			Name:  "commit_sha",
			Value: tag.Commit.ID,
		})
	}
	return metadata
}
