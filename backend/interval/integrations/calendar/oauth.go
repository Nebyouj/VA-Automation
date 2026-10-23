package calendar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	calendarScope = "https://www.googleapis.com/auth/calendar.events"
	sheetsScope   = "https://www.googleapis.com/auth/spreadsheets"
)

type OAuthManager struct {
	config    *oauth2.Config
	tokenPath string

	mu     sync.Mutex
	states map[string]time.Time
}

func NewOAuthManager(
	clientID string,
	clientSecret string,
	redirectURL string,
	tokenPath string,
) *OAuthManager {
	return &OAuthManager{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				calendarScope,
				sheetsScope,
			},
			Endpoint: google.Endpoint,
		},
		tokenPath: tokenPath,
		states:    make(map[string]time.Time),
	}
}

// StartHandler redirects the user to Google's consent screen.
func (m *OAuthManager) StartHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, "Failed to generate OAuth state", http.StatusInternalServerError)
		return
	}

	state := hex.EncodeToString(stateBytes)

	m.mu.Lock()
	now := time.Now()
	for existingState, expiry := range m.states {
		if now.After(expiry) {
			delete(m.states, existingState)
		}
	}
	m.states[state] = now.Add(10 * time.Minute)
	m.mu.Unlock()

	url := m.config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CallbackHandler exchanges Google's authorization code for a token.
func (m *OAuthManager) CallbackHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	state := r.URL.Query().Get("state")

	m.mu.Lock()
	expiry, exists := m.states[state]
	if exists {
		delete(m.states, state)
	}
	m.mu.Unlock()

	if !exists || time.Now().After(expiry) {
		http.Error(w, "Invalid or expired OAuth state", http.StatusBadRequest)
		return
	}

	if providerError := r.URL.Query().Get("error"); providerError != "" {
		http.Error(w, "Google authorization was denied", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	token, err := m.config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange authorization code", http.StatusBadGateway)
		return
	}

	if err := m.saveToken(token); err != nil {
		http.Error(w, "Failed to save authorization token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "Google Calendar authorization successful. You can return to the application.")
}

// LoadToken loads the saved token and checks that it is still usable.
// oauth2.Config.Client automatically refreshes expired access tokens
// when a refresh token is available.
func (m *OAuthManager) LoadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(m.tokenPath)
	if err != nil {
		return nil, fmt.Errorf("read Google Calendar token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("decode Google Calendar token: %w", err)
	}

	if token.RefreshToken == "" && !token.Valid() {
		return nil, errors.New("Google Calendar token expired and has no refresh token; authorize again")
	}

	return &token, nil
}

func (m *OAuthManager) saveToken(token *oauth2.Token) error {
	dir := filepath.Dir(m.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}

	tempFile, err := os.CreateTemp(dir, ".calendar-token-*")
	if err != nil {
		return fmt.Errorf("create temporary token file: %w", err)
	}

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := tempFile.Chmod(0600); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("secure temporary token file: %w", err)
	}

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write token: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close token file: %w", err)
	}

	if err := os.Rename(tempPath, m.tokenPath); err != nil {
		return fmt.Errorf("save token file: %w", err)
	}

	return nil
}
