//
// Copyright 2021, Sander van Harmelen
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
	"fmt"
	"net/http"
)

type (
	ApplicationsServiceInterface interface {
		CreateApplication(opt *CreateApplicationOptions, options ...RequestOptionFunc) (*Application, *Response, error)
		ListApplications(opt *ListApplicationsOptions, options ...RequestOptionFunc) ([]*Application, *Response, error)
		DeleteApplication(application int, options ...RequestOptionFunc) (*Response, error)
	}

	// ApplicationsService handles communication with administrables applications
	// of the Gitlab API.
	//
	// Gitlab API docs: https://docs.gitlab.com/api/applications/
	ApplicationsService struct {
		client *Client
	}
)

var _ ApplicationsServiceInterface = (*ApplicationsService)(nil)

// Application represents a GitLab application
type Application struct {
	ID              int    `json:"id"`
	ApplicationID   string `json:"application_id"`
	ApplicationName string `json:"application_name"`
	Secret          string `json:"secret"`
	CallbackURL     string `json:"callback_url"`
	Confidential    bool   `json:"confidential"`
}

// CreateApplicationOptions represents the available CreateApplication() options.
//
// GitLab API docs:
// https://docs.gitlab.com/api/applications/#create-an-application
type CreateApplicationOptions struct {
	Name         *string `url:"name,omitempty" json:"name,omitempty"`
	RedirectURI  *string `url:"redirect_uri,omitempty" json:"redirect_uri,omitempty"`
	Scopes       *string `url:"scopes,omitempty" json:"scopes,omitempty"`
	Confidential *bool   `url:"confidential,omitempty" json:"confidential,omitempty"`
}

// CreateApplication creates a new application owned by the authenticated user.
//
// Gitlab API docs: https://docs.gitlab.com/api/applications/#create-an-application
func (s *ApplicationsService) CreateApplication(opt *CreateApplicationOptions, options ...RequestOptionFunc) (*Application, *Response, error) {
	req, err := s.client.NewRequest(http.MethodPost, "applications", opt, options)
	if err != nil {
		return nil, nil, err
	}

	a := new(Application)
	resp, err := s.client.Do(req, a)
	if err != nil {
		return nil, resp, err
	}

	return a, resp, nil
}

// ListApplicationsOptions represents the available
// ListApplications() options.
type ListApplicationsOptions ListOptions

// ListApplications get a list of administrables applications by the authenticated user
//
// Gitlab API docs : https://docs.gitlab.com/api/applications/#list-all-applications
func (s *ApplicationsService) ListApplications(opt *ListApplicationsOptions, options ...RequestOptionFunc) ([]*Application, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "applications", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var as []*Application
	resp, err := s.client.Do(req, &as)
	if err != nil {
		return nil, resp, err
	}

	return as, resp, nil
}

// DeleteApplication removes a specific application.
//
// GitLab API docs:
// https://docs.gitlab.com/api/applications/#delete-an-application
func (s *ApplicationsService) DeleteApplication(application int, options ...RequestOptionFunc) (*Response, error) {
	u := fmt.Sprintf("applications/%d", application)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
