package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"idp/internal/authentication"
)

const (
	sessionCookieSecure = "__Host-idp_session"
	csrfCookieSecure    = "__Host-idp_csrf"
	loginCSRFSecure     = "__Host-idp_login_csrf"
	sessionCookieDev    = "idp_session"
	csrfCookieDev       = "idp_csrf"
	loginCSRFDev        = "idp_login_csrf"
	csrfHeader          = "X-CSRF-Token"
)

type authContextKey struct{}

type requestAuth struct {
	Session   *authentication.AuthenticatedSession
	Principal authentication.Principal
}

func PrincipalFromContext(ctx context.Context) (authentication.Principal, bool) {
	auth, ok := ctx.Value(authContextKey{}).(requestAuth)
	return auth.Principal, ok && auth.Session != nil
}

func (s *Server) authenticationMiddleware(next http.Handler) http.Handler {
	if s.Auth == nil {
		if s.disableAuthenticationForTests {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "Authentication service is unavailable.", http.StatusServiceUnavailable)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicAuthPath(r) {
			next.ServeHTTP(w, r)
			return
		}
		session, hadCookie, err := s.authenticateRequest(r)
		if err != nil {
			writeError(w, err)
			return
		}
		if session == nil {
			if hadCookie {
				s.clearAuthCookies(w)
			}
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeJSON(w, http.StatusUnauthorized, problemBody("UNAUTHENTICATED", "Authentication is required."))
				return
			}
			returnTo := authentication.SafeReturnTo(r.URL.RequestURI())
			http.Redirect(w, r, "/ui/login?return_to="+url.QueryEscape(returnTo), http.StatusSeeOther)
			return
		}
		if isUnsafeMethod(r.Method) && !validSessionCSRF(r, session.CSRFHash) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeJSON(w, http.StatusForbidden, problemBody("INVALID_CSRF", "The form or session has expired. Reload and try again."))
			} else {
				http.Error(w, "The form or session has expired. Reload and try again.", http.StatusForbidden)
			}
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, requestAuth{Session: session, Principal: session.Principal()})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicAuthPath(r *http.Request) bool {
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
		(r.URL.Path == "/ui/login" || strings.HasPrefix(r.URL.Path, "/ui/assets/")) {
		return true
	}
	if r.Method == http.MethodGet && r.URL.Path == "/api/auth/login-context" {
		return true
	}
	if r.Method == http.MethodPost && r.URL.Path == "/api/auth/login" {
		return true
	}
	return false
}

func (s *Server) authenticateRequest(r *http.Request) (*authentication.AuthenticatedSession, bool, error) {
	cookie, err := r.Cookie(s.authCookieNames().session)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	session, err := s.Auth.Authenticate(r.Context(), cookie.Value)
	if errors.Is(err, authentication.ErrInvalidCredentials) {
		return nil, true, nil
	}
	return session, true, err
}

func (s *Server) apiLoginContext(w http.ResponseWriter, r *http.Request) {
	returnTo := authentication.SafeReturnTo(r.URL.Query().Get("return_to"))
	if session, hadCookie, err := s.authenticateRequest(r); err != nil {
		writeError(w, err)
		return
	} else if session != nil {
		writeJSON(w, http.StatusOK, map[string]string{"redirectTo": returnTo})
		return
	} else if hadCookie {
		s.clearAuthCookies(w)
	}
	raw, err := randomToken()
	if err != nil {
		writeError(w, err)
		return
	}
	names := s.authCookieNames()
	http.SetCookie(w, &http.Cookie{Name: names.loginCSRF, Value: raw, Path: "/", MaxAge: 600,
		HttpOnly: true, Secure: !s.AuthDevelopmentCookies, SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]string{"csrfToken": raw, "returnTo": returnTo})
}

func (s *Server) apiLogin(w http.ResponseWriter, r *http.Request) {
	if !s.sameOriginRequest(r) {
		writeJSON(w, http.StatusForbidden, problemBody("INVALID_ORIGIN", "The login request origin is not allowed."))
		return
	}
	names := s.authCookieNames()
	cookie, err := r.Cookie(names.loginCSRF)
	if err != nil || !constantTokenEqual(cookie.Value, r.Header.Get(csrfHeader)) {
		writeJSON(w, http.StatusForbidden, problemBody("INVALID_CSRF", "The sign-in form has expired. Reload and try again."))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		ReturnTo string `json:"returnTo"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, problemBody("INVALID_INPUT", "Invalid sign-in request."))
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, problemBody("INVALID_INPUT", "Invalid sign-in request."))
		return
	}
	material, err := s.Auth.SignIn(r.Context(), body.Username, body.Password, requestSource(r), body.ReturnTo)
	if err != nil {
		var limited *authentication.RateLimitedError
		switch {
		case errors.Is(err, authentication.ErrInvalidCredentials):
			writeJSON(w, http.StatusUnauthorized, problemBody("INVALID_CREDENTIALS", "The username or password is incorrect."))
		case errors.As(err, &limited):
			seconds := int64(limited.RetryAfter.Round(time.Second) / time.Second)
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
			writeJSON(w, http.StatusTooManyRequests, problemBody("RATE_LIMITED", "Too many sign-in attempts. Try again later."))
		default:
			writeError(w, err)
		}
		return
	}
	s.setSessionCookies(w, material)
	s.clearCookie(w, names.loginCSRF, true)
	writeJSON(w, http.StatusOK, map[string]string{"redirectTo": material.RedirectTo})
}

func (s *Server) apiLogout(w http.ResponseWriter, r *http.Request) {
	auth, _ := r.Context().Value(authContextKey{}).(requestAuth)
	if auth.Session == nil {
		writeJSON(w, http.StatusUnauthorized, problemBody("UNAUTHENTICATED", "Authentication is required."))
		return
	}
	if err := s.Auth.SignOut(r.Context(), auth.Session.TokenHash); err != nil {
		writeError(w, err)
		return
	}
	s.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func validSessionCSRF(r *http.Request, expectedHash []byte) bool {
	raw := r.Header.Get(csrfHeader)
	if raw == "" {
		if err := r.ParseForm(); err == nil {
			raw = r.Form.Get("csrf_token")
		}
	}
	hash := sha256.Sum256([]byte(raw))
	return raw != "" && subtle.ConstantTimeCompare(hash[:], expectedHash) == 1
}

func (s *Server) sameOriginRequest(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	expectedScheme := "https"
	if s.AuthDevelopmentCookies {
		expectedScheme = "http"
	}
	return err == nil && strings.EqualFold(u.Scheme, expectedScheme) && strings.EqualFold(u.Host, r.Host)
}

func requestSource(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func isUnsafeMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

type cookieNames struct{ session, csrf, loginCSRF string }

func (s *Server) authCookieNames() cookieNames {
	if s.AuthDevelopmentCookies {
		return cookieNames{sessionCookieDev, csrfCookieDev, loginCSRFDev}
	}
	return cookieNames{sessionCookieSecure, csrfCookieSecure, loginCSRFSecure}
}

func (s *Server) setSessionCookies(w http.ResponseWriter, material *authentication.SessionMaterial) {
	names := s.authCookieNames()
	secure := !s.AuthDevelopmentCookies
	http.SetCookie(w, &http.Cookie{Name: names.session, Value: material.SessionToken, Path: "/", MaxAge: int(authentication.SessionLifetime.Seconds()),
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: names.csrf, Value: material.CSRFToken, Path: "/", MaxAge: int(authentication.SessionLifetime.Seconds()),
		Secure: secure, SameSite: http.SameSiteLaxMode})
}

func (s *Server) clearAuthCookies(w http.ResponseWriter) {
	names := s.authCookieNames()
	s.clearCookie(w, names.session, true)
	s.clearCookie(w, names.csrf, false)
}

func (s *Server) clearCookie(w http.ResponseWriter, name string, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0),
		HttpOnly: httpOnly, Secure: !s.AuthDevelopmentCookies, SameSite: http.SameSiteLaxMode})
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate CSRF token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func constantTokenEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aHash := sha256.Sum256([]byte(a))
	bHash := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(aHash[:], bHash[:]) == 1
}

func problemBody(code, message string) map[string]any {
	return map[string]any{"problems": []map[string]string{{"code": code, "message": message}}}
}
