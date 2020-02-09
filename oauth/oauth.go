// package oauth is based on https://support.google.com/transitpartners/answer/2529132?hl=en&ref_topic=2527461.
package oauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/amwolff/google-gtfs-realtime-tools/provider"
	"github.com/golang/protobuf/proto"
)

const (
	DefaultTokenExchangeURL = "https://accounts.google.com/o/oauth2/token"
	DefaultFeedUploadURL    = "https://partnerdash.google.com/push-upload"
)

type clientSecret struct {
	Installed struct {
		AuthProviderX509CertURL string   `json:"auth_provider_x509_cert_url"`
		AuthURI                 string   `json:"auth_uri"`
		ClientEmail             string   `json:"client_email"`
		ClientID                string   `json:"client_id"`
		ClientSecret            string   `json:"client_secret"`
		ClientX509CertURL       string   `json:"client_x509_cert_url"`
		RedirectURIs            []string `json:"redirect_uris"`
		TokenURI                string   `json:"token_uri"`
	} `json:"installed"`
}

type tokenData struct {
	AccessToken    string    `json:"access_token"`
	ExpirationDate time.Time `json:"expiration_date"`
	TokenType      string    `json:"token_type"`
	RefreshToken   string    `json:"refresh_token"`
}

type Client struct {
	httpClient       *http.Client
	secret           clientSecret
	tokens           tokenData
	tokenExchangeURL string
	cachePath        string
	feedUploadURL    string
}

type exchangeResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

func doExchange(
	tokenExchangeURL string,
	form io.Reader,
	contentType string,
	httpClient *http.Client) (tokenData, error) {

	req, err := http.NewRequest(http.MethodPost, tokenExchangeURL, form)
	if err != nil {
		return tokenData{}, fmt.Errorf("NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	now := time.Now()
	res, err := httpClient.Do(req)
	if err != nil {
		return tokenData{}, fmt.Errorf("Do: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return tokenData{}, fmt.Errorf("non-200 status: %s", res.Status)
	}

	var xchRes exchangeResponse
	if err := json.NewDecoder(res.Body).Decode(&xchRes); err != nil {
		return tokenData{}, fmt.Errorf("Decode: %w", err)
	}

	ret := tokenData{
		AccessToken:    xchRes.AccessToken,
		ExpirationDate: now.Add(time.Duration(xchRes.ExpiresIn) * time.Second),
		TokenType:      xchRes.TokenType,
		RefreshToken:   xchRes.RefreshToken,
	}

	return ret, nil
}

type FeedMessageWrapper struct {
	Name string
	File io.Reader
}

func WrapperFromProvider(name string, feedProvider provider.FeedMessageProvider) (
	FeedMessageWrapper,
	error) {

	m, err := feedProvider.Provide()
	if err != nil {
		return FeedMessageWrapper{}, fmt.Errorf("Provide: %w", err)
	}
	b, err := proto.Marshal(m)
	if err != nil {
		return FeedMessageWrapper{}, fmt.Errorf("Marshal: %w", err)
	}
	return FeedMessageWrapper{
		Name: name,
		File: bytes.NewReader(b),
	}, nil
}

// TODO: this could be done a little bit better (?):
//       1. Create better wrapper type for gtfs-realtime feed (??)
//       2. Use mime/multipart.Form instead of map[string]interface{} (???)
//       But since it's not really exported I'm leaving this the way it is.
func createRFC2388Form(values map[string]interface{}) (io.Reader, string, error) {
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	for k, v := range values {
		switch x := v.(type) {
		case string:
			if err := w.WriteField(k, x); err != nil {
				return nil, "", fmt.Errorf("WriteField: %w", err)
			}
		case FeedMessageWrapper:
			f, err := w.CreateFormFile(k, x.Name)
			if err != nil {
				return nil, "", fmt.Errorf("CreateFormFile: %w", err)
			}
			if _, err := io.Copy(f, x.File); err != nil {
				return nil, "", fmt.Errorf("Copy: %w", err)
			}
		default:
			return nil, "", errors.New("unsupported value type")
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("Close: %w", err)
	}
	return b, w.FormDataContentType(), nil
}

func exchangeForTokens(
	authorizationCode string,
	secret clientSecret,
	tokenExchangeURL string,
	httpClient *http.Client) (tokenData, error) {

	form, contentType, err := createRFC2388Form(map[string]interface{}{
		"code":          authorizationCode,
		"client_id":     secret.Installed.ClientID,
		"client_secret": secret.Installed.ClientSecret,
		"redirect_uri":  secret.Installed.RedirectURIs[0],
		"grant_type":    "authorization_code",
	})
	if err != nil {
		return tokenData{}, fmt.Errorf("createRFC2388Form: %w", err)
	}

	tokens, err := doExchange(tokenExchangeURL, form, contentType, httpClient)
	if err != nil {
		return tokens, fmt.Errorf("doExchange: %w", err)
	}

	return tokens, nil
}

func writeTokensToFile(tokens tokenData, path string) error {
	b, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("Marshal: %w", err)
	}
	if err := ioutil.WriteFile(path, b, 0600); err != nil {
		return fmt.Errorf("WriteFile: %w", err)
	}
	return nil
}

func NewClient(
	httpClient *http.Client,
	clientSecretJSON io.Reader,
	tokensCachePath,
	tokenExchangeURL,
	authorizationCode,
	feedUploadURL string) (*Client, error) {

	if len(tokenExchangeURL) == 0 || len(tokensCachePath) == 0 {
		return nil, errors.New("CachePath/ExchangeURL must not be empty")
	}

	var secret clientSecret
	if err := json.NewDecoder(clientSecretJSON).Decode(&secret); err != nil {
		return nil, fmt.Errorf("Decode: %w", err)
	}

	cleanCachePath := filepath.Clean(tokensCachePath)
	if _, err := os.Stat(cleanCachePath); !os.IsNotExist(err) {
		// No authorization needed - fast path.
		b, err := ioutil.ReadFile(cleanCachePath)
		if err != nil {
			return nil, fmt.Errorf("ReadFile: %w", err)
		}
		var tokens tokenData
		if err := json.Unmarshal(b, &tokens); err != nil {
			return nil, fmt.Errorf("Unmarshal: %w", err)
		}
		return &Client{
			httpClient:       httpClient,
			secret:           secret,
			tokens:           tokens,
			tokenExchangeURL: tokenExchangeURL,
			cachePath:        cleanCachePath,
			feedUploadURL:    feedUploadURL,
		}, nil
	}

	tokens, err := exchangeForTokens(
		authorizationCode,
		secret,
		tokenExchangeURL,
		httpClient)
	if err != nil {
		return nil, fmt.Errorf("exchangeForTokens: %w", err)
	}

	if err := writeTokensToFile(tokens, cleanCachePath); err != nil {
		return nil, fmt.Errorf("writeTokensToFile: %w", err)
	}

	return &Client{
		httpClient:       httpClient,
		secret:           secret,
		tokens:           tokens,
		tokenExchangeURL: tokenExchangeURL,
		cachePath:        cleanCachePath,
		feedUploadURL:    feedUploadURL,
	}, nil
}

func (c Client) IsAccessTokenExpired() bool {
	return time.Now().After(c.tokens.ExpirationDate)
}

func (c *Client) MaybeRefreshAccessToken() error {
	if !c.IsAccessTokenExpired() {
		return nil
	}

	form, contentType, err := createRFC2388Form(map[string]interface{}{
		"client_id":     c.secret.Installed.ClientID,
		"client_secret": c.secret.Installed.ClientSecret,
		"refresh_token": c.tokens.RefreshToken,
		"grant_type":    "refresh_token",
	})

	tokens, err := doExchange(
		c.tokenExchangeURL,
		form,
		contentType,
		c.httpClient)
	if err != nil {
		return fmt.Errorf("doExchange: %w", err)
	}

	if len(tokens.RefreshToken) == 0 { // TODO: most likely drop this branch.
		tokens.RefreshToken = c.tokens.RefreshToken
	}

	if err := writeTokensToFile(tokens, c.cachePath); err != nil {
		return fmt.Errorf("writeTokensToFile: %w", err)
	}

	c.tokens = tokens

	return nil
}

func getBearer(token string) string {
	return fmt.Sprintf("Bearer %s", token)
}

func (c *Client) UploadFeedMessage(
	alkaliAccountID, realtimeFeedID string,
	wrapper FeedMessageWrapper) error {

	form, contentType, err := createRFC2388Form(map[string]interface{}{
		"alkali_application_name": "transit",
		"alkali_account_id":       alkaliAccountID,
		"alkali_upload_type":      "realtime_push_upload",
		"alkali_application_id":   "100003100",
		"realtime_feed_id":        realtimeFeedID,
		"file":                    wrapper,
	})

	req, err := http.NewRequest(http.MethodPost, c.feedUploadURL, form)
	if err != nil {
		return fmt.Errorf("NewRequest: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", getBearer(c.tokens.AccessToken))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Do: %w", err)
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("ReadAll: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %q", res.Status, content)
	}

	return nil
}
