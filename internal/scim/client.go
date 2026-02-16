// Package scim implements a SCIM 2.0 client for pushing users and groups
// to Houston's /v1/scim/v2/okta endpoint.
package scim

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"oauth-tester/internal/store"
)

// Client makes SCIM 2.0 HTTP calls to a Houston instance.
type Client struct {
	BaseURL    string // e.g. https://houston.example.com
	AuthCode   string
	HTTPClient *http.Client
	LogStore   store.LogStore // optional, logs requests/responses when set
}

const scimPath = "/v1/scim/v2/okta"

// SCIM resource types.

type SCIMUser struct {
	Schemas     []string        `json:"schemas,omitempty"`
	ID          string          `json:"id,omitempty"`
	UserName    string          `json:"userName"`
	DisplayName string          `json:"displayName,omitempty"`
	Name        *SCIMName       `json:"name,omitempty"`
	Emails      []SCIMEmail     `json:"emails,omitempty"`
	Active      bool            `json:"active"`
	Groups      []SCIMGroupRef  `json:"groups,omitempty"`
	Meta        *SCIMMeta       `json:"meta,omitempty"`
}

type SCIMName struct {
	GivenName  string `json:"givenName,omitempty"`
	FamilyName string `json:"familyName,omitempty"`
}

type SCIMEmail struct {
	Primary bool   `json:"primary"`
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
}

type SCIMGroupRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
}

type SCIMGroup struct {
	Schemas     []string          `json:"schemas,omitempty"`
	ID          string            `json:"id,omitempty"`
	DisplayName string            `json:"displayName"`
	Members     []SCIMMemberRef   `json:"members,omitempty"`
	Meta        *SCIMMeta         `json:"meta,omitempty"`
}

type SCIMMemberRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
}

type SCIMMeta struct {
	ResourceType string `json:"resourceType,omitempty"`
}

type SCIMListResponse struct {
	Schemas      []string          `json:"schemas"`
	TotalResults int               `json:"totalResults"`
	StartIndex   int               `json:"startIndex"`
	ItemsPerPage int               `json:"itemsPerPage"`
	Resources    json.RawMessage   `json:"Resources"`
}

type SCIMPatchOp struct {
	Schemas    []string        `json:"schemas"`
	Operations []SCIMOperation `json:"Operations"`
}

type SCIMOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) do(method, path string, body interface{}) ([]byte, int, error) {
	u := strings.TrimRight(c.BaseURL, "/") + path

	var reqBodyBytes []byte
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request: %w", err)
		}
		reqBodyBytes = b
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/scim+json")
	req.Header.Set("Accept", "application/scim+json")

	// Houston Basic auth: base64-encode the auth code as-is.
	encoded := base64.StdEncoding.EncodeToString([]byte(c.AuthCode))
	req.Header.Set("Authorization", "Basic "+encoded)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		c.logEntry(method, path, reqBodyBytes, nil, 0, req.Header, nil)
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	c.logEntry(method, path, reqBodyBytes, respBody, resp.StatusCode, req.Header, resp.Header)
	return respBody, resp.StatusCode, nil
}

func (c *Client) logEntry(method, path string, reqBody, respBody []byte, status int, reqHeaders, respHeaders http.Header) {
	if c.LogStore == nil {
		return
	}

	// Parse path and query from the full path string.
	p := path
	q := ""
	if i := strings.IndexByte(path, '?'); i >= 0 {
		p = path[:i]
		q = path[i+1:]
	}

	entry := &store.LogEntry{
		Timestamp:       time.Now().UTC(),
		Source:          "scim",
		Method:          method,
		Path:            p,
		Query:           q,
		RequestHeaders:  headerString(reqHeaders),
		RequestBody:     prettyJSON(reqBody),
		ResponseStatus:  status,
		ResponseHeaders: headerString(respHeaders),
		ResponseBody:    prettyJSON(respBody),
	}
	c.LogStore.SaveLog(entry)
}

func headerString(h http.Header) string {
	if h == nil {
		return ""
	}
	var b strings.Builder
	for k, vals := range h {
		for _, v := range vals {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func prettyJSON(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		return string(b)
	}
	return buf.String()
}

// ListUsers fetches all SCIM users from Houston, handling pagination.
func (c *Client) ListUsers() ([]SCIMUser, error) {
	var all []SCIMUser
	startIndex := 1
	count := 100

	for {
		params := url.Values{
			"startIndex": {fmt.Sprintf("%d", startIndex)},
			"count":      {fmt.Sprintf("%d", count)},
		}
		body, status, err := c.do("GET", scimPath+"/Users?"+params.Encode(), nil)
		if err != nil {
			return nil, err
		}
		if status != 200 {
			return nil, fmt.Errorf("list users: HTTP %d: %s", status, body)
		}

		var list SCIMListResponse
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("decode list response: %w", err)
		}

		var users []SCIMUser
		if err := json.Unmarshal(list.Resources, &users); err != nil {
			return nil, fmt.Errorf("decode users: %w", err)
		}
		all = append(all, users...)

		if len(all) >= list.TotalResults || len(users) == 0 {
			break
		}
		startIndex += len(users)
	}
	return all, nil
}

// CreateUser creates a new SCIM user in Houston.
func (c *Client) CreateUser(email, displayName string) (*SCIMUser, error) {
	user := SCIMUser{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		UserName:    email,
		DisplayName: displayName,
		Emails: []SCIMEmail{
			{Primary: true, Value: email, Type: "work"},
		},
		Active: true,
	}

	body, status, err := c.do("POST", scimPath+"/Users", user)
	if err != nil {
		return nil, err
	}
	if status != 201 && status != 200 {
		return nil, fmt.Errorf("create user %s: HTTP %d: %s", email, status, body)
	}

	var created SCIMUser
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("decode created user: %w", err)
	}
	return &created, nil
}

// UpdateUser does a full PUT replacement on a SCIM user.
func (c *Client) UpdateUser(id, email, displayName string) error {
	user := SCIMUser{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		UserName:    email,
		DisplayName: displayName,
		Emails: []SCIMEmail{
			{Primary: true, Value: email, Type: "work"},
		},
		Active: true,
	}

	body, status, err := c.do("PUT", scimPath+"/Users/"+url.PathEscape(id), user)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("update user %s: HTTP %d: %s", email, status, body)
	}
	return nil
}

// DeleteUser deactivates a SCIM user in Houston (soft delete).
func (c *Client) DeleteUser(id string) error {
	body, status, err := c.do("DELETE", scimPath+"/Users/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	// Okta handler returns 204 on success.
	if status != 204 && status != 200 {
		return fmt.Errorf("delete user %s: HTTP %d: %s", id, status, body)
	}
	return nil
}

// ListGroups fetches all SCIM groups from Houston, handling pagination.
func (c *Client) ListGroups() ([]SCIMGroup, error) {
	var all []SCIMGroup
	startIndex := 1
	count := 100

	for {
		params := url.Values{
			"startIndex": {fmt.Sprintf("%d", startIndex)},
			"count":      {fmt.Sprintf("%d", count)},
		}
		body, status, err := c.do("GET", scimPath+"/Groups?"+params.Encode(), nil)
		if err != nil {
			return nil, err
		}
		if status != 200 {
			return nil, fmt.Errorf("list groups: HTTP %d: %s", status, body)
		}

		var list SCIMListResponse
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("decode list response: %w", err)
		}

		var groups []SCIMGroup
		if err := json.Unmarshal(list.Resources, &groups); err != nil {
			return nil, fmt.Errorf("decode groups: %w", err)
		}
		all = append(all, groups...)

		if len(all) >= list.TotalResults || len(groups) == 0 {
			break
		}
		startIndex += len(groups)
	}
	return all, nil
}

// CreateGroup creates a new SCIM group in Houston.
func (c *Client) CreateGroup(displayName string) (*SCIMGroup, error) {
	group := SCIMGroup{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		DisplayName: displayName,
	}

	body, status, err := c.do("POST", scimPath+"/Groups", group)
	if err != nil {
		return nil, err
	}
	if status != 201 && status != 200 {
		return nil, fmt.Errorf("create group %s: HTTP %d: %s", displayName, status, body)
	}

	var created SCIMGroup
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, fmt.Errorf("decode created group: %w", err)
	}
	return &created, nil
}

// DeleteGroup hard-deletes a SCIM group in Houston.
func (c *Client) DeleteGroup(id string) error {
	body, status, err := c.do("DELETE", scimPath+"/Groups/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	if status != 204 && status != 200 {
		return fmt.Errorf("delete group %s: HTTP %d: %s", id, status, body)
	}
	return nil
}

// PatchGroupMembers adds or removes members from a SCIM group.
func (c *Client) PatchGroupMembers(groupID string, addUserIDs, removeUserIDs []string) error {
	if len(addUserIDs) == 0 && len(removeUserIDs) == 0 {
		return nil
	}

	patch := SCIMPatchOp{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
	}

	if len(addUserIDs) > 0 {
		members := make([]map[string]string, len(addUserIDs))
		for i, uid := range addUserIDs {
			members[i] = map[string]string{"value": uid}
		}
		patch.Operations = append(patch.Operations, SCIMOperation{
			Op:    "add",
			Path:  "members",
			Value: members,
		})
	}

	for _, uid := range removeUserIDs {
		patch.Operations = append(patch.Operations, SCIMOperation{
			Op:   "remove",
			Path: fmt.Sprintf("members[value eq \"%s\"]", uid),
		})
	}

	body, status, err := c.do("PATCH", scimPath+"/Groups/"+url.PathEscape(groupID), patch)
	if err != nil {
		return err
	}
	if status != 200 && status != 204 {
		return fmt.Errorf("patch group %s members: HTTP %d: %s", groupID, status, body)
	}
	return nil
}
