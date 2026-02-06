//
// Copyright 2023, Nick Westbury
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"net/http"
	"time"
)

type (
	ProjectRepositoryStorageMoveServiceInterface interface {
		RetrieveAllStorageMoves(opts RetrieveAllProjectStorageMovesOptions, options ...RequestOptionFunc) ([]*ProjectRepositoryStorageMove, *Response, error)
		RetrieveAllStorageMovesForProject(project int64, opts RetrieveAllProjectStorageMovesOptions, options ...RequestOptionFunc) ([]*ProjectRepositoryStorageMove, *Response, error)
		GetStorageMove(repositoryStorage int64, options ...RequestOptionFunc) (*ProjectRepositoryStorageMove, *Response, error)
		GetStorageMoveForProject(project int64, repositoryStorage int64, options ...RequestOptionFunc) (*ProjectRepositoryStorageMove, *Response, error)
		ScheduleStorageMoveForProject(project int64, opts ScheduleStorageMoveForProjectOptions, options ...RequestOptionFunc) (*ProjectRepositoryStorageMove, *Response, error)
		ScheduleAllStorageMoves(opts ScheduleAllProjectStorageMovesOptions, options ...RequestOptionFunc) (*Response, error)
	}

	// ProjectRepositoryStorageMoveService handles communication with the
	// repositories related methods of the GitLab API.
	//
	// GitLab API docs:
	// https://docs.gitlab.com/api/project_repository_storage_moves/
	ProjectRepositoryStorageMoveService struct {
		client *Client
	}
)

var _ ProjectRepositoryStorageMoveServiceInterface = (*ProjectRepositoryStorageMoveService)(nil)

// ProjectRepositoryStorageMove represents the status of a repository move.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/
type ProjectRepositoryStorageMove struct {
	ID                     int64              `json:"id"`
	CreatedAt              *time.Time         `json:"created_at"`
	State                  string             `json:"state"`
	SourceStorageName      string             `json:"source_storage_name"`
	DestinationStorageName string             `json:"destination_storage_name"`
	Project                *RepositoryProject `json:"project"`
}

type RepositoryProject struct {
	ID                int64      `json:"id"`
	Description       string     `json:"description"`
	Name              string     `json:"name"`
	NameWithNamespace string     `json:"name_with_namespace"`
	Path              string     `json:"path"`
	PathWithNamespace string     `json:"path_with_namespace"`
	CreatedAt         *time.Time `json:"created_at"`
}

// RetrieveAllProjectStorageMovesOptions represents the available
// RetrieveAllStorageMoves() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#retrieve-all-project-repository-storage-moves
type RetrieveAllProjectStorageMovesOptions struct {
	ListOptions
}

// RetrieveAllStorageMoves retrieves all project repository storage moves
// accessible by the authenticated user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#retrieve-all-project-repository-storage-moves
func (p ProjectRepositoryStorageMoveService) RetrieveAllStorageMoves(opts RetrieveAllProjectStorageMovesOptions, options ...RequestOptionFunc) ([]*ProjectRepositoryStorageMove, *Response, error) {
	return do[[]*ProjectRepositoryStorageMove](p.client,
		withPath("project_repository_storage_moves"),
		withAPIOpts(opts),
		withRequestOpts(options...),
	)
}

// RetrieveAllStorageMovesForProject retrieves all repository storage moves for
// a single project accessible by the authenticated user.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#retrieve-all-repository-storage-moves-for-a-project
func (p ProjectRepositoryStorageMoveService) RetrieveAllStorageMovesForProject(project int64, opts RetrieveAllProjectStorageMovesOptions, options ...RequestOptionFunc) ([]*ProjectRepositoryStorageMove, *Response, error) {
	return do[[]*ProjectRepositoryStorageMove](p.client,
		withPath("projects/%d/repository_storage_moves", project),
		withAPIOpts(opts),
		withRequestOpts(options...),
	)
}

// GetStorageMove gets a single project repository storage move.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#get-a-single-project-repository-storage-move
func (p ProjectRepositoryStorageMoveService) GetStorageMove(repositoryStorage int64, options ...RequestOptionFunc) (*ProjectRepositoryStorageMove, *Response, error) {
	return do[*ProjectRepositoryStorageMove](p.client,
		withPath("project_repository_storage_moves/%d", repositoryStorage),
		withRequestOpts(options...),
	)
}

// GetStorageMoveForProject gets a single repository storage move for a project.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#get-a-single-repository-storage-move-for-a-project
func (p ProjectRepositoryStorageMoveService) GetStorageMoveForProject(project int64, repositoryStorage int64, options ...RequestOptionFunc) (*ProjectRepositoryStorageMove, *Response, error) {
	return do[*ProjectRepositoryStorageMove](p.client,
		withPath("projects/%d/repository_storage_moves/%d", project, repositoryStorage),
		withRequestOpts(options...),
	)
}

// ScheduleStorageMoveForProjectOptions represents the available
// ScheduleStorageMoveForProject() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#schedule-a-repository-storage-move-for-a-project
type ScheduleStorageMoveForProjectOptions struct {
	DestinationStorageName *string `url:"destination_storage_name,omitempty" json:"destination_storage_name,omitempty"`
}

// ScheduleStorageMoveForProject schedule a repository to be moved for a project.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#schedule-a-repository-storage-move-for-a-project
func (p ProjectRepositoryStorageMoveService) ScheduleStorageMoveForProject(project int64, opts ScheduleStorageMoveForProjectOptions, options ...RequestOptionFunc) (*ProjectRepositoryStorageMove, *Response, error) {
	return do[*ProjectRepositoryStorageMove](p.client,
		withMethod(http.MethodPost),
		withPath("projects/%d/repository_storage_moves", project),
		withAPIOpts(opts),
		withRequestOpts(options...),
	)
}

// ScheduleAllProjectStorageMovesOptions represents the available
// ScheduleAllStorageMoves() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#schedule-repository-storage-moves-for-all-projects-on-a-storage-shard
type ScheduleAllProjectStorageMovesOptions struct {
	SourceStorageName      *string `url:"source_storage_name,omitempty" json:"source_storage_name,omitempty"`
	DestinationStorageName *string `url:"destination_storage_name,omitempty" json:"destination_storage_name,omitempty"`
}

// ScheduleAllStorageMoves schedules all repositories to be moved.
//
// GitLab API docs:
// https://docs.gitlab.com/api/project_repository_storage_moves/#schedule-repository-storage-moves-for-all-projects-on-a-storage-shard
func (p ProjectRepositoryStorageMoveService) ScheduleAllStorageMoves(opts ScheduleAllProjectStorageMovesOptions, options ...RequestOptionFunc) (*Response, error) {
	_, resp, err := do[none](p.client,
		withMethod(http.MethodPost),
		withPath("project_repository_storage_moves"),
		withAPIOpts(opts),
		withRequestOpts(options...),
	)
	return resp, err
}
