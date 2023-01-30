package batch

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is used for making GraphQL requests via HTTP using pre-configured
// url, query, header and login information.
//
// Client is thread-safe.
type Client struct {
	mu         sync.RWMutex
	httpClient *http.Client
	query      string
	url        string
	header     http.Header
	oauth      *OAuthConfig
	token      *string
}

// NewClient creates a new Client using the provided configuration and query.
func NewClient(config Config, query string) (*Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = config.Connections
	transport.MaxIdleConns = config.Connections
	transport.MaxIdleConnsPerHost = config.Connections

	header := http.Header{}
	if len(config.Headers) != 0 {
		hs := strings.Join(config.Headers, "\r\n") + "\r\n\r\n"
		r := textproto.NewReader(bufio.NewReader(strings.NewReader(hs)))
		mimeHeader, err := r.ReadMIMEHeader()
		if err != nil {
			return nil, err
		}
		header = http.Header(mimeHeader)
	}

	client := &Client{
		httpClient: &http.Client{
			Transport: transport,
		},
		query:  query,
		url:    config.URL,
		header: header,
	}
	if config.OAuth.URL != "" {
		client.oauth = &config.OAuth
	}
	if config.BearerToken != "" {
		client.token = &config.BearerToken
	}

	return client, nil
}

// Do sends the GraphQL request using the configured query and the provided
// variables.
//
// For successful requests it will return the response body and no error.
// For unsuccessful requests it will return an error and if possible the
// response body.
//
// If the client was configured with OAuth credentials, it will follow a
// client_credentials flow to receive a valid authorization token. The token
// will automatically be renewed before its expiry.
//
// For any authorization other flow you can provide the required authorization
// headers during client creation.
func (c *Client) Do(variables map[string]any) (io.ReadCloser, error) {
	token, err := c.ensureValidToken()
	if err != nil {
		return nil, err
	}

	requestBody, err := json.Marshal(requestParameters{
		Query:     c.query,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", c.url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	for hk, hv := range c.header {
		for _, v := range hv {
			request.Header.Add(hk, v)
		}
	}

	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return response.Body, fmt.Errorf("invalid status code %v", response.StatusCode)
	}

	return response.Body, nil
}

func (c *Client) ensureValidToken() (string, error) {
	if c.oauth == nil {
		if c.token == nil {
			return "", nil
		}
		return *c.token, nil
	}
	token := ""
	c.mu.RLock()
	if c.token != nil {
		token = *c.token
	}
	c.mu.RUnlock()
	if token != "" {
		return token, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != nil {
		return *c.token, nil
	}
	token, err := c.login()
	if err != nil {
		return "", err
	}
	c.token = &token
	return token, nil
}

func (c *Client) login() (string, error) {
	data := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {c.oauth.Scope},
	}
	request, err := http.NewRequest("POST", c.oauth.URL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(c.oauth.ClientID, c.oauth.ClientSecret)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		return "", fmt.Errorf("invalid status code %v during login", response.StatusCode)
	}
	resp := &struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}{}
	err = json.NewDecoder(response.Body).Decode(resp)
	if err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("login response did not include access token")
	}
	// ensure that token is removed before it expires
	expiresIn90p := time.Duration(math.Floor(float64(resp.ExpiresIn)*0.9)) * time.Second
	time.AfterFunc(expiresIn90p, func() {
		c.mu.Lock()
		c.token = nil
		c.mu.Unlock()
	})
	return resp.AccessToken, nil
}

type requestParameters struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}
