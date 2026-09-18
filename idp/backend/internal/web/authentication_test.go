package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"idp/internal/authentication"
)

type webAuthRepository struct {
	credential *authentication.CredentialRecord
	session    *authentication.SessionRecord
	revoked    bool
}

func (r *webAuthRepository) FindCredential(_ context.Context, username string) (*authentication.CredentialRecord, error) {
	if r.credential != nil && r.credential.Username == username {
		copy := *r.credential
		return &copy, nil
	}
	return nil, nil
}

func (*webAuthRepository) BlockedUntil(context.Context, []byte, []byte, time.Time) (time.Time, error) {
	return time.Time{}, nil
}

func (*webAuthRepository) RecordFailure(context.Context, string, []byte, int, time.Time, time.Duration) (time.Time, error) {
	return time.Time{}, nil
}

func (r *webAuthRepository) CreateSessionAndClearFailures(_ context.Context, session authentication.SessionRecord, _ []byte) error {
	r.session = &session
	return nil
}

func (r *webAuthRepository) AuthenticateSession(_ context.Context, tokenHash []byte, now, idleCutoff time.Time) (*authentication.AuthenticatedSession, error) {
	if r.session == nil || r.revoked || !bytes.Equal(tokenHash, r.session.TokenHash) ||
		!r.session.ExpiresAt.After(now) || !r.session.LastSeenAt.After(idleCutoff) {
		return nil, nil
	}
	r.session.LastSeenAt = now
	return &authentication.AuthenticatedSession{SessionID: r.session.SessionID, UserID: r.session.UserID,
		Username: r.credential.Username, DisplayName: r.credential.DisplayName,
		CSRFHash: append([]byte(nil), r.session.CSRFHash...)}, nil
}

func (r *webAuthRepository) RevokeSession(_ context.Context, tokenHash []byte, _ time.Time) error {
	if r.session != nil && bytes.Equal(tokenHash, r.session.TokenHash) {
		r.revoked = true
	}
	return nil
}

func (*webAuthRepository) CreateLocalUser(context.Context, string, string, string, time.Time) error {
	return nil
}

func (*webAuthRepository) ResetLocalPassword(context.Context, string, string, time.Time) error {
	return nil
}

func (*webAuthRepository) SetLocalUserStatus(context.Context, string, string, time.Time) error {
	return nil
}

func testWebAuth(t *testing.T) (*authentication.Service, *webAuthRepository) {
	t.Helper()
	hash, err := (authentication.PasswordHasher{}).Hash("a sufficiently long password")
	if err != nil {
		t.Fatal(err)
	}
	repo := &webAuthRepository{credential: &authentication.CredentialRecord{UserID: "user-1", Username: "developer",
		DisplayName: "Developer", Status: "ACTIVE", PasswordHash: hash}}
	service, err := authentication.NewService(repo, bytes.Repeat([]byte{0x24}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return service, repo
}

func cookieNamed(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found in %#v", name, cookies)
	return nil
}

func TestAuthenticationMiddlewareSeparatesBrowserAndAPI(t *testing.T) {
	service, _ := testWebAuth(t)
	handler := (&Server{Auth: service, AuthDevelopmentCookies: true}).Handler()

	api := httptest.NewRecorder()
	handler.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/application-definitions", nil))
	if api.Code != http.StatusUnauthorized || api.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("protected API response = %d %q %q", api.Code, api.Header().Get("Content-Type"), api.Body.String())
	}

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/ui/applications?tab=mine", nil))
	if page.Code != http.StatusSeeOther || page.Header().Get("Location") != "/ui/login?return_to=%2Fui%2Fapplications%3Ftab%3Dmine" {
		t.Fatalf("protected page response = %d location %q", page.Code, page.Header().Get("Location"))
	}
}

func TestServerFailsClosedWithoutAuthenticationService(t *testing.T) {
	response := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ui/applications", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestLoginContextSignInAndLogout(t *testing.T) {
	service, repo := testWebAuth(t)
	handler := (&Server{Auth: service, AuthDevelopmentCookies: true}).Handler()

	contextResponse := httptest.NewRecorder()
	handler.ServeHTTP(contextResponse, httptest.NewRequest(http.MethodGet, "/api/auth/login-context?return_to=%2Fui%2Fapplications%2Fnew", nil))
	if contextResponse.Code != http.StatusOK {
		t.Fatalf("login context response = %d %q", contextResponse.Code, contextResponse.Body.String())
	}
	loginCSRF := cookieNamed(t, contextResponse.Result().Cookies(), loginCSRFDev)
	var loginContext map[string]string
	if err := json.Unmarshal(contextResponse.Body.Bytes(), &loginContext); err != nil {
		t.Fatal(err)
	}
	if loginContext["csrfToken"] != loginCSRF.Value || loginContext["returnTo"] != "/ui/applications/new" {
		t.Fatalf("login context = %#v", loginContext)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "http://idp.local/api/auth/login",
		bytes.NewBufferString(`{"username":"developer","password":"a sufficiently long password","returnTo":"/ui/applications/new"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.Header.Set("Origin", "http://idp.local")
	loginRequest.Header.Set("Sec-Fetch-Site", "same-origin")
	loginRequest.Header.Set(csrfHeader, loginCSRF.Value)
	loginRequest.AddCookie(loginCSRF)
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK || repo.session == nil {
		t.Fatalf("login response = %d %q", loginResponse.Code, loginResponse.Body.String())
	}
	sessionCookie := cookieNamed(t, loginResponse.Result().Cookies(), sessionCookieDev)
	csrfCookie := cookieNamed(t, loginResponse.Result().Cookies(), csrfCookieDev)
	if !sessionCookie.HttpOnly || sessionCookie.Secure || csrfCookie.HttpOnly || csrfCookie.Secure {
		t.Fatalf("development cookies = %#v", loginResponse.Result().Cookies())
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "http://idp.local/api/auth/logout", nil)
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.AddCookie(csrfCookie)
	logoutRequest.Header.Set(csrfHeader, csrfCookie.Value)
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent || !repo.revoked {
		t.Fatalf("logout response = %d revoked=%v body=%q", logoutResponse.Code, repo.revoked, logoutResponse.Body.String())
	}
	if cookieNamed(t, logoutResponse.Result().Cookies(), sessionCookieDev).MaxAge != -1 {
		t.Fatal("logout did not clear the session cookie")
	}
}

func TestLoginRejectsMissingOriginAndCSRF(t *testing.T) {
	service, repo := testWebAuth(t)
	handler := (&Server{Auth: service, AuthDevelopmentCookies: true}).Handler()
	request := httptest.NewRequest(http.MethodPost, "http://idp.local/api/auth/login",
		bytes.NewBufferString(`{"username":"developer","password":"a sufficiently long password"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || repo.session != nil {
		t.Fatalf("login response = %d session=%#v body=%q", response.Code, repo.session, response.Body.String())
	}
}

func TestSameOriginRequestUsesCookieProfileScheme(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://idp.local/api/auth/login", nil)
	request.Header.Set("Origin", "http://idp.local")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	if !(&Server{AuthDevelopmentCookies: true}).sameOriginRequest(request) {
		t.Fatal("development profile rejected its HTTP origin")
	}
	if (&Server{}).sameOriginRequest(request) {
		t.Fatal("secure profile accepted an HTTP origin")
	}
	request.Header.Set("Origin", "https://idp.local")
	if !(&Server{}).sameOriginRequest(request) {
		t.Fatal("secure profile rejected its HTTPS origin")
	}
}

func TestAuthenticatedUnsafeRequestRequiresSessionCSRF(t *testing.T) {
	service, repo := testWebAuth(t)
	material, err := service.SignIn(context.Background(), "developer", "a sufficiently long password", "127.0.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	handler := (&Server{Auth: service, AuthDevelopmentCookies: true}).Handler()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieDev, Value: material.SessionToken})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || repo.revoked {
		t.Fatalf("logout response = %d revoked=%v body=%q", response.Code, repo.revoked, response.Body.String())
	}
}

func TestAuthenticationMiddlewareAttachesProviderNeutralPrincipal(t *testing.T) {
	service, _ := testWebAuth(t)
	material, err := service.SignIn(context.Background(), "developer", "a sufficiently long password", "127.0.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Auth: service, AuthDevelopmentCookies: true}
	handler := server.authenticationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok || principal.UserID != "user-1" || principal.Username != "developer" || principal.DisplayName != "Developer" {
			t.Errorf("principal = %#v, present=%v", principal, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieDev, Value: material.SessionToken})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("response status = %d", response.Code)
	}
}
