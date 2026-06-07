package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionManager_PublicMethods(t *testing.T) {
	manager := NewSessionManager([]byte("test-secret"), time.Hour, false)

	recorder := httptest.NewRecorder()
	if err := manager.Set(recorder); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %+v", cookies)
	}
	if cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly || cookies[0].Secure {
		t.Fatalf("unexpected session cookie: %+v", cookies[0])
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookies[0])
	if !manager.Valid(request) {
		t.Fatalf("expected session cookie to be valid")
	}

	otherManager := NewSessionManager([]byte("other-secret"), time.Hour, false)
	if otherManager.Valid(request) {
		t.Fatalf("expected session cookie signed with another secret to be invalid")
	}

	clearRecorder := httptest.NewRecorder()
	manager.Clear(clearRecorder)
	clearCookies := clearRecorder.Result().Cookies()
	if len(clearCookies) != 1 || clearCookies[0].MaxAge != -1 {
		t.Fatalf("unexpected clear cookie: %+v", clearCookies)
	}
}

func TestRequireSession(t *testing.T) {
	manager := NewSessionManager([]byte("test-secret"), time.Hour, false)
	called := false
	handler := RequireSession(manager, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/", nil))
	if unauthorized.Code != http.StatusUnauthorized || called {
		t.Fatalf("unexpected unauthorized result: status=%d called=%v", unauthorized.Code, called)
	}

	login := httptest.NewRecorder()
	if err := manager.Set(login); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	authorizedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	authorizedRequest.AddCookie(login.Result().Cookies()[0])

	authorized := httptest.NewRecorder()
	handler.ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusNoContent || !called {
		t.Fatalf("unexpected authorized result: status=%d called=%v", authorized.Code, called)
	}
}

func TestRequireDeviceToken(t *testing.T) {
	tokens := map[string]struct{}{"device-token": {}}
	called := false
	handler := RequireDeviceToken(tokens, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/", nil))
	if unauthorized.Code != http.StatusUnauthorized || called {
		t.Fatalf("unexpected unauthorized result: status=%d called=%v", unauthorized.Code, called)
	}

	headerRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	headerRequest.Header.Set("X-Device-Token", "device-token")
	headerRecorder := httptest.NewRecorder()
	handler.ServeHTTP(headerRecorder, headerRequest)
	if headerRecorder.Code != http.StatusNoContent || !called {
		t.Fatalf("unexpected header token result: status=%d called=%v", headerRecorder.Code, called)
	}

	called = false
	bearerRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	bearerRequest.Header.Set("Authorization", "Bearer device-token")
	bearerRecorder := httptest.NewRecorder()
	handler.ServeHTTP(bearerRecorder, bearerRequest)
	if bearerRecorder.Code != http.StatusNoContent || !called {
		t.Fatalf("unexpected bearer token result: status=%d called=%v", bearerRecorder.Code, called)
	}
}
