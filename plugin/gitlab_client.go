// Copyright 2021 Splunk Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitlabtoken

import (
	"fmt"
	"time"

	"github.com/xanzy/go-gitlab"
)

const (
	clientTTL = 30 * time.Minute
)

type Client interface {
	// ListProjectAccessToken retrieves a list of project access tokens for a given project ID
	ListProjectAccessToken(int) ([]*PAT, error)
	// CreateProjectAccessToken create access token for given a project ID
	CreateProjectAccessToken(*BaseTokenStorageEntry, *time.Time) (*PAT, error)
	// RevokeProjectAccessToken revokes the access token
	RevokeProjectAccessToken(*BaseTokenStorageEntry) error
	Valid() bool
}

type gitlabClient struct {
	client     *gitlab.Client
	expiration time.Time
}

var _ Client = &gitlabClient{}

func NewClient(config *ConfigStorageEntry) (Client, error) {
	if config == nil {
		return nil, fmt.Errorf("gitlab backend configuration has not been set up")
	}
	gc := &gitlabClient{
		expiration: time.Now().Add(clientTTL),
	}

	opt := gitlab.WithBaseURL(config.BaseURL)
	if config.Token == "" {
		return nil, fmt.Errorf("token isn't configured")
	}
	c, err := gitlab.NewClient(config.Token, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gitlab client iwht endpoint %s: %v", config.BaseURL, err)
	}
	gc.client = c

	return gc, nil
}

func (gc *gitlabClient) Valid() bool {
	return gc != nil && time.Now().Before(gc.expiration)
}

func (gc *gitlabClient) ListProjectAccessToken(pid int) ([]*PAT, error) {
	// Use the GitLab API client to list project access tokens
	tokens, _, err := gc.client.ProjectAccessTokens.ListProjectAccessTokens(pid, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list project access tokens for project ID %d: %w", pid, err)
	}

	// Convert the tokens to the PAT type
	var result []*PAT
	for _, token := range tokens {
		result = append(result, &PAT{
			ID:          token.ID,
			Name:        token.Name,
			Scopes:      token.Scopes,
			ExpiresAt:   token.ExpiresAt,
			AccessLevel: token.AccessLevel,
		})
	}

	return result, nil
}

func (gc *gitlabClient) CreateProjectAccessToken(tokenStorage *BaseTokenStorageEntry, expiresAt *time.Time) (*PAT, error) {
	opt := gitlab.CreateProjectAccessTokenOptions{
		Name:   &tokenStorage.Name,
		Scopes: &tokenStorage.Scopes,
	}
	if expiresAt != nil {
		expiration := gitlab.ISOTime(*expiresAt)
		opt.ExpiresAt = &expiration
	}
	if tokenStorage.AccessLevel != 0 {
		opt.AccessLevel = (*gitlab.AccessLevelValue)(&tokenStorage.AccessLevel)
	}
	pat, _, err := gc.client.ProjectAccessTokens.CreateProjectAccessToken(tokenStorage.ID, &opt)
	if err != nil {
		return nil, err
	}
	return pat, nil
}

func (gc *gitlabClient) RevokeProjectAccessToken(tokenStorage *BaseTokenStorageEntry) error {
	// Use the GitLab API client to revoke the project access token
	_, err := gc.client.ProjectAccessTokens.RevokeProjectAccessToken(tokenStorage.ID, tokenStorage.TokenID)
	if err != nil {
		return fmt.Errorf("failed to revoke project access token with ID %d for project ID %d: %w", tokenStorage.TokenID, tokenStorage.ID, err)
	}

	return nil
}
