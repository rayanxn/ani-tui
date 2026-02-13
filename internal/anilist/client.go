package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const graphqlEndpoint = "https://graphql.anilist.co"

// Client is a GraphQL client for the AniList API.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a new AniList client. Token may be empty for public queries.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{},
	}
}

// graphqlRequest is the JSON body sent to the AniList GraphQL endpoint.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphqlResponse wraps the raw JSON response.
type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// doQuery executes a GraphQL query and unmarshals the data field into target.
func (c *Client) doQuery(ctx context.Context, query string, vars map[string]any, target any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: vars})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AniList API error (status %d): %s", resp.StatusCode, respBody)
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("AniList: %s", gqlResp.Errors[0].Message)
	}

	if target != nil {
		if err := json.Unmarshal(gqlResp.Data, target); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}

	return nil
}

// SearchAnime searches for anime by name.
func (c *Client) SearchAnime(ctx context.Context, search string, page int) ([]Media, error) {
	var result struct {
		Page struct {
			PageInfo PageInfo `json:"pageInfo"`
			Media    []Media  `json:"media"`
		} `json:"Page"`
	}

	vars := map[string]any{
		"search": search,
		"page":   page,
	}

	if err := c.doQuery(ctx, searchAnimeQuery, vars, &result); err != nil {
		return nil, err
	}

	return result.Page.Media, nil
}

// GetAnimeDetails retrieves full details for a specific anime.
func (c *Client) GetAnimeDetails(ctx context.Context, id int) (Media, error) {
	var result struct {
		Media Media `json:"Media"`
	}

	vars := map[string]any{"id": id}

	if err := c.doQuery(ctx, getAnimeDetailsQuery, vars, &result); err != nil {
		return Media{}, err
	}

	return result.Media, nil
}

// GetUserList retrieves a user's anime list.
func (c *Client) GetUserList(ctx context.Context, userID int) (MediaListCollection, error) {
	var result struct {
		MediaListCollection MediaListCollection `json:"MediaListCollection"`
	}

	vars := map[string]any{"userId": userID}

	if err := c.doQuery(ctx, getUserListQuery, vars, &result); err != nil {
		return MediaListCollection{}, err
	}

	return result.MediaListCollection, nil
}

// UpdateProgress updates the watch progress for an anime.
func (c *Client) UpdateProgress(ctx context.Context, mediaID, progress int, status string) error {
	vars := map[string]any{
		"mediaId":  mediaID,
		"progress": progress,
		"status":   status,
	}

	return c.doQuery(ctx, updateProgressMutation, vars, nil)
}

// GetViewer retrieves the authenticated user's information.
func (c *Client) GetViewer(ctx context.Context) (User, error) {
	var result struct {
		Viewer User `json:"Viewer"`
	}

	if err := c.doQuery(ctx, viewerQuery, nil, &result); err != nil {
		return User{}, err
	}

	return result.Viewer, nil
}
