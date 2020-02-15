package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const gigabyte = 1 << (10 * 3)

func mustLoadClientSecretsJSON() clientSecret {
	b, err := ioutil.ReadFile(filepath.Clean("./testdata/client_secrets.json"))
	if err != nil {
		panic(fmt.Sprintf("ReadFile: %v", err))
	}
	var ret clientSecret
	if err := json.Unmarshal(b, &ret); err != nil {
		panic(fmt.Sprintf("Unmarshal: %v", err))
	}
	return ret
}

func checkExchangeRequestCorrectness(
	t *testing.T,
	r *http.Request,
	code string,
	secret clientSecret) {

	// Check form correctness for this particular request.
	if err := r.ParseMultipartForm(gigabyte); err != nil {
		panic(fmt.Sprintf("ParseMultipartForm: %v", err))
	}
	assert.Equal(t, []string{code}, r.MultipartForm.Value["code"])
	assert.Equal(
		t,
		[]string{secret.Installed.ClientID},
		r.MultipartForm.Value["client_id"])
	assert.Equal(
		t,
		[]string{secret.Installed.ClientSecret},
		r.MultipartForm.Value["client_secret"])
	assert.Equal(
		t,
		[]string{secret.Installed.RedirectURIs[0]},
		r.MultipartForm.Value["redirect_uri"])
	assert.Equal(
		t,
		[]string{"authorization_code"},
		r.MultipartForm.Value["grant_type"])
}

func getExchangeForTokensHandler(
	t *testing.T,
	code string,
	secret clientSecret) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		checkExchangeRequestCorrectness(t, r, code, secret)
		// Return desired response.
		fmt.Fprint(w, `{"access_token":"1/fFAGRNJru1FTz70BzhT3Zg","expires_in"`+
			`:3920,"token_type":"Bearer","refresh_token":"1/xEoDL4iW3cxlI7yDbS`+
			`RFYNG01kVKM2C-259HOF2aQbI"}`)
	}
}

func getNewClientHandler(
	t *testing.T,
	code string,
	secret clientSecret) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		checkExchangeRequestCorrectness(t, r, code, secret)
		// Return desired response.
		fmt.Fprint(w, `{"access_token":"9cddb84a-5ab4-4ee4-9abc-cd53183b45bd",`+
			`"expires_in":1337,"token_type":"Bearer","refresh_token":"ed93fc15`+
			`-6ff6-4efc-9c75-faccde6925fe"}`)
	}
}

func getRefreshAccessTokenHandler(
	t *testing.T,
	secret clientSecret) http.HandlerFunc {

	var called bool
	return func(w http.ResponseWriter, r *http.Request) {
		// Ensure this handler is called only once in a test.
		assert.False(t, called)
		called = true

		// Check form correctness for this particular request.
		if err := r.ParseMultipartForm(gigabyte); err != nil {
			panic(fmt.Sprintf("ParseMultipartForm: %v", err))
		}
		assert.Equal(
			t,
			[]string{secret.Installed.ClientID},
			r.MultipartForm.Value["client_id"])
		assert.Equal(
			t,
			[]string{secret.Installed.ClientSecret},
			r.MultipartForm.Value["client_secret"])
		assert.Equal(
			t,
			[]string{"f472681a-051f-4671-8df5-afffd9d6c47f"},
			r.MultipartForm.Value["refresh_token"])
		assert.Equal(
			t,
			[]string{"refresh_token"},
			r.MultipartForm.Value["grant_type"])

		// Return desired response.
		fmt.Fprint(w, `{"access_token":"1/fFAGRNJru1FTz70BzhT3Zg","expires_in"`+
			`:3920,"token_type":"Bearer"}`)
	}
}

func getUploadFeedMessageHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check form correctness for this particular request.
		if err := r.ParseMultipartForm(gigabyte); err != nil {
			panic(fmt.Sprintf("ParseMultipartForm: %v", err))
		}
		assert.Equal(t, []string{"transit"}, r.MultipartForm.Value["alkali_application_name"])
		assert.Equal(t, []string{""}, r.MultipartForm.Value["alkali_account_id"])
		assert.Equal(t, []string{""}, r.MultipartForm.Value["alkali_upload_type"])
		assert.Equal(t, []string{""}, r.MultipartForm.Value["alkali_application_id"])
		assert.Equal(t, []string{""}, r.MultipartForm.Value["realtime_feed_id"])

		// TODO verify more things
	}
}

func mustLoadCachedTokens(path string) tokenData {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("ReadFile: %v", err))
	}
	var ret tokenData
	if err := json.Unmarshal(b, &ret); err != nil {
		panic(fmt.Sprintf("Unmarshal: %v", err))
	}
	return ret
}

func checkTokensEquality(t *testing.T, expected, actual tokenData) {
	assert.Equal(t, expected.AccessToken, actual.AccessToken)
	assert.Equal(t, expected.TokenType, actual.TokenType)
	assert.Equal(t, expected.RefreshToken, actual.RefreshToken)
	// Check ExpirationDate with to-second tolerance.
	assert.Equal(
		t,
		expected.ExpirationDate.Unix(),
		actual.ExpirationDate.Unix())
}

func TestCreateRFC2388Form(t *testing.T) {
	f, err := os.Open(filepath.Clean("./testdata/trip-updates-full.asciipb"))
	if err != nil {
		panic(fmt.Sprintf("Open: %v", err))
	}
	defer f.Close()

	testBytes, err := ioutil.ReadAll(f)
	if err != nil {
		panic(fmt.Sprintf("ReadAll: %v", err))
	}

	wrapper := FeedMessageWrapper{
		Name: filepath.Base(f.Name()),
		File: bytes.NewReader(testBytes),
	}

	form, contentType, err := createRFC2388Form(map[string]interface{}{
		"alkali_application_name": "transit",
		"alkali_account_id":       "a60924bd-dcfd-4f95-9914-ee28b5484d37",
		"alkali_upload_type":      "realtime_push_upload",
		"alkali_application_id":   "100003100",
		"realtime_feed_id":        "2f182302-da25-4562-aa14-7890da496693",
		"file":                    wrapper,
	})
	assert.NoError(t, err)

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		panic(fmt.Sprintf("ParseMediaType: %v", err))
	}

	r := multipart.NewReader(form, params["boundary"])

	parsed, err := r.ReadForm(gigabyte)
	if err != nil {
		panic(fmt.Sprintf("ReadForm: %v", err))
	}

	assert.Equal(
		t,
		[]string{"transit"},
		parsed.Value["alkali_application_name"])
	assert.Equal(
		t,
		[]string{"a60924bd-dcfd-4f95-9914-ee28b5484d37"},
		parsed.Value["alkali_account_id"])
	assert.Equal(
		t,
		[]string{"realtime_push_upload"},
		parsed.Value["alkali_upload_type"])
	assert.Equal(
		t,
		[]string{"100003100"},
		parsed.Value["alkali_application_id"])
	assert.Equal(
		t,
		[]string{"2f182302-da25-4562-aa14-7890da496693"},
		parsed.Value["realtime_feed_id"])
	assert.Equal(
		t,
		"trip-updates-full.asciipb",
		parsed.File["file"][0].Filename)

	formFile, err := parsed.File["file"][0].Open()
	if err != nil {
		panic(fmt.Sprintf("Open: %v", err))
	}
	defer formFile.Close()

	formBytes, err := ioutil.ReadAll(formFile)
	if err != nil {
		panic(fmt.Sprintf("ReadAll: %v", err))
	}

	assert.Equal(t, testBytes, formBytes)
}

func TestExchangeForTokens(t *testing.T) {
	const code = "18dba4ce-8110-4513-840f-b57a96c93705"

	secret := mustLoadClientSecretsJSON()

	ts := httptest.NewTLSServer(getExchangeForTokensHandler(t, code, secret))
	defer ts.Close()

	tokens, err := exchangeForTokens(code, secret, ts.URL, ts.Client())
	assert.NoError(t, err)

	assert.Equal(t, "1/fFAGRNJru1FTz70BzhT3Zg", tokens.AccessToken)
	assert.False(t, tokens.ExpirationDate.Before(time.Now()))
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Equal(t, "1/xEoDL4iW3cxlI7yDbSRFYNG01kVKM2C-259HOF2aQbI", tokens.RefreshToken)
}

func TestWriteTokensToFile(t *testing.T) {
	tokensPath := filepath.Clean("/tmp/9ce8d8af-41a5-41bb-bfd7-a6dc85b0b6eb")

	for i := 0; i < 24; i++ {
		tokens := tokenData{
			AccessToken:    "3163948a-3624-4ab6-b198-afe6ca88216a",
			ExpirationDate: time.Date(1949, time.June, 8, i, i, i, i, time.UTC),
			TokenType:      "Bearer",
			RefreshToken:   "2fd78b4d-42b6-440a-b5e8-efc685a52ce7",
		}
		assert.NoError(t, writeTokensToFile(tokens, tokensPath))
		assert.Equal(t, tokens, mustLoadCachedTokens(tokensPath))
	}

	// Cleanup.
	if err := os.Remove(tokensPath); err != nil {
		panic(fmt.Sprintf("Remove: %v", err))
	}
}

func TestNewClient(t *testing.T) {
	const code = "1b6790db-23cf-49c1-8747-cbb41a4a2b8d"

	secret := mustLoadClientSecretsJSON()

	ts := httptest.NewTLSServer(getNewClientHandler(t, code, secret))
	defer ts.Close()

	secretFile, err := os.Open(filepath.Clean("./testdata/client_secrets.json"))
	if err != nil {
		panic(fmt.Sprintf("Open: %v", err))
	}
	defer secretFile.Close()

	tsClient := ts.Client()
	tokensPath := "/tmp/293553b1-7333-4fd8-9f41-32b710121bd0"

	client, err := NewClient(
		tsClient,
		secretFile,
		tokensPath,
		ts.URL,
		code,
		DefaultFeedUploadURL)
	assert.NoError(t, err)

	tokens := tokenData{
		AccessToken:    "9cddb84a-5ab4-4ee4-9abc-cd53183b45bd",
		ExpirationDate: time.Now().Add(1337 * time.Second),
		TokenType:      "Bearer",
		RefreshToken:   "ed93fc15-6ff6-4efc-9c75-faccde6925fe",
	}

	cleanTokensPath := filepath.Clean(tokensPath)

	// Assert internal state.
	assert.Equal(t, tsClient, client.httpClient)
	assert.Equal(t, secret, client.secret)
	assert.Equal(t, ts.URL, client.tokenExchangeURL)
	assert.Equal(t, cleanTokensPath, client.cachePath)
	assert.Equal(t, DefaultFeedUploadURL, client.feedUploadURL)

	// Assert internal state: test internal token storage and cache.
	checkTokensEquality(t, tokens, mustLoadCachedTokens(cleanTokensPath))
	checkTokensEquality(t, tokens, client.tokens)

	// Cleanup.
	if err := os.Remove(cleanTokensPath); err != nil {
		panic(fmt.Sprintf("Remove: %v", err))
	}
}

func TestNewClientFastPath(t *testing.T) {
	const code = "043715ac-cdbb-45d3-98f6-aa73cf9e32fc"

	secret := mustLoadClientSecretsJSON()

	ts := httptest.NewTLSServer(getNewClientHandler(t, code, secret))
	defer ts.Close()

	secretFile, err := os.Open(filepath.Clean("./testdata/client_secrets.json"))
	if err != nil {
		panic(fmt.Sprintf("Open: %v", err))
	}
	defer secretFile.Close()

	tsClient := ts.Client()
	tokensPath := "./testdata/test_tokens"

	client, err := NewClient(
		tsClient,
		secretFile,
		tokensPath,
		ts.URL,
		code,
		DefaultFeedUploadURL)
	assert.NoError(t, err)

	// Assert internal state.
	assert.Equal(t, tsClient, client.httpClient)
	assert.Equal(t, secret, client.secret)
	assert.Equal(t, ts.URL, client.tokenExchangeURL)
	assert.Equal(t, filepath.Clean(tokensPath), client.cachePath)
	assert.Equal(t, DefaultFeedUploadURL, client.feedUploadURL)

	l, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		panic(fmt.Sprintf("LoadLocation: %v", err))
	}
	// 2020-02-07T15:04:48.870053479+01:00
	expirationDate := time.Date(2020, time.February, 7, 15, 4, 48, 870053479, l)

	tokens := tokenData{
		AccessToken:    "b65dce0c-c66f-4c08-83ad-803f451e0a26",
		ExpirationDate: expirationDate,
		TokenType:      "Bearer",
		RefreshToken:   "a30ae706-8fcb-46cd-80c4-9ce66fa5fe7e",
	}

	// Assert internal state: test internal token storage.
	checkTokensEquality(t, tokens, client.tokens)
}

func TestClient_IsAccessTokenExpired(t *testing.T) {
	cs, err := ioutil.ReadFile(filepath.Clean("./testdata/client_secrets.json"))
	if err != nil {
		panic(fmt.Sprintf("Open: %v", err))
	}

	tokensPath := "/tmp/5f2f8f32-6983-4c70-8e15-40acc5d15fde"
	cleanTokensPath := filepath.Clean(tokensPath)

	if err := ioutil.WriteFile(
		cleanTokensPath,
		[]byte(
			fmt.Sprintf(
				`{"expiration_date":"%s"}`,
				time.Now().Add(time.Minute).Format(time.RFC3339Nano))),
		0777); err != nil {

		panic(fmt.Sprintf("WriteFile: %v", err))
	}

	client, err := NewClient(
		nil,
		bytes.NewReader(cs),
		tokensPath,
		DefaultTokenExchangeURL,
		"",
		"")
	assert.NoError(t, err)

	assert.False(t, client.IsAccessTokenExpired())

	clientWithExpired, err := NewClient(
		nil,
		bytes.NewReader(cs),
		"./testdata/test_tokens_expired",
		DefaultTokenExchangeURL,
		"",
		"")
	assert.NoError(t, err)

	assert.True(t, clientWithExpired.IsAccessTokenExpired())

	// Cleanup.
	if err := os.Remove(cleanTokensPath); err != nil {
		panic(fmt.Sprintf("Remove: %v", err))
	}
}

func TestClient_MaybeRefreshAccessToken(t *testing.T) {
	secret := mustLoadClientSecretsJSON()

	ts := httptest.NewTLSServer(getRefreshAccessTokenHandler(t, secret))
	defer ts.Close()

	secretFile, err := os.Open(filepath.Clean("./testdata/client_secrets.json"))
	if err != nil {
		panic(fmt.Sprintf("Open: %v", err))
	}
	defer secretFile.Close()

	tokensPath := "/tmp/613d878d-6ff3-4940-a93b-9c20f33e8cca"
	cleanTokensPath := filepath.Clean(tokensPath)

	if err := ioutil.WriteFile(
		cleanTokensPath,
		[]byte(`{"access_token":"6018652e-b34b-493d-9e11-a741cc07e637","expira`+
			`tion_date":"1970-01-01T00:00:00Z","token_type":"Bearer","refresh_`+
			`token":"f472681a-051f-4671-8df5-afffd9d6c47f"}`),
		0777); err != nil {

		panic(fmt.Sprintf("WriteFile: %v", err))
	}

	tsClient := ts.Client()

	client, err := NewClient(
		ts.Client(),
		secretFile,
		tokensPath,
		ts.URL,
		"",
		DefaultFeedUploadURL)
	assert.NoError(t, err)

	assert.NoError(t, client.MaybeRefreshAccessToken())

	tokens := tokenData{
		AccessToken:    "1/fFAGRNJru1FTz70BzhT3Zg",
		ExpirationDate: time.Now().Add(3920 * time.Second),
		TokenType:      "Bearer",
		RefreshToken:   "f472681a-051f-4671-8df5-afffd9d6c47f",
	}

	// Assert internal state.
	assert.Equal(t, tsClient, client.httpClient)
	assert.Equal(t, secret, client.secret)
	assert.Equal(t, ts.URL, client.tokenExchangeURL)
	assert.Equal(t, cleanTokensPath, client.cachePath)
	assert.Equal(t, DefaultFeedUploadURL, client.feedUploadURL)

	// Assert internal state: test internal token storage and cache.
	checkTokensEquality(t, tokens, mustLoadCachedTokens(cleanTokensPath))
	checkTokensEquality(t, tokens, client.tokens)

	// Make sure IsAccessTokenExpired is properly called.
	for i := 0; i < 100; i++ {
		assert.NoError(t, client.MaybeRefreshAccessToken())
	}

	// Cleanup.
	if err := os.Remove(cleanTokensPath); err != nil {
		panic(fmt.Sprintf("Remove: %v", err))
	}
}

func TestClient_UploadFeedMessage(t *testing.T) {
	ts := httptest.NewTLSServer(getUploadFeedMessageHandler(t))
	defer ts.Close()

	secretFile, err := os.Open(filepath.Clean("./testdata/client_secrets.json"))
	if err != nil {
		panic(fmt.Sprintf("Open: %v", err))
	}
	defer secretFile.Close()

	tsClient := ts.Client()
	tokensPath := "./testdata/test_tokens"

	client, err := NewClient(
		tsClient,
		secretFile,
		tokensPath,
		DefaultTokenExchangeURL,
		"",
		ts.URL)
	assert.NoError(t, err)

	// TODO proceed with this code.
}
