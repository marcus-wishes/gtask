package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"gtask/internal/config"
	"gtask/internal/exitcode"
	"gtask/internal/service"
)

const (
	// OAuth scope for Google Tasks
	tasksScope = "https://www.googleapis.com/auth/tasks"

	// OAuth callback timeout
	oauthCallbackTimeout = 5 * time.Minute

	// Token exchange timeout
	tokenExchangeTimeout = 30 * time.Second

	// Starting port for OAuth callback server
	oauthStartPort = 8085

	// Max port attempts
	oauthMaxPortAttempts = 5
)

func init() {
	Register(&LoginCmd{})
}

// LoginCmd implements the login command.
type LoginCmd struct{}

func (c *LoginCmd) Name() string      { return "login" }
func (c *LoginCmd) Aliases() []string { return nil }
func (c *LoginCmd) Synopsis() string  { return "Authenticate with Google" }
func (c *LoginCmd) Usage() string     { return "gtask login [common flags]" }
func (c *LoginCmd) NeedsAuth() bool   { return false }

func (c *LoginCmd) RegisterFlags(fs *flag.FlagSet) {}

func (c *LoginCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	// Check if oauth_client.json exists
	if !cfg.HasOAuthClient() {
		fmt.Fprintf(errOut, "error: oauth_client.json not found in %s\n", cfg.Dir)
		return exitcode.AuthError
	}

	// Check if already logged in (token exists and is valid)
	if cfg.HasToken() {
		if isTokenValid(cfg.TokenPath()) {
			if !cfg.Quiet {
				fmt.Fprintln(out, "already logged in")
			}
			return exitcode.Success
		}
	}

	// Load OAuth client config
	clientJSON, err := os.ReadFile(cfg.OAuthClientPath())
	if err != nil {
		fmt.Fprintf(errOut, "error: failed to read oauth_client.json: %v\n", err)
		return exitcode.AuthError
	}

	oauthConfig, err := google.ConfigFromJSON(clientJSON, tasksScope)
	if err != nil {
		fmt.Fprintf(errOut, "error: invalid oauth_client.json: %v\n", err)
		return exitcode.AuthError
	}

	// Find available port
	port, listener, err := findAvailablePort()
	if err != nil {
		fmt.Fprintf(errOut, "error: could not bind to local port for OAuth callback\n")
		return exitcode.AuthError
	}
	defer listener.Close()

	// Set redirect URL
	redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)
	oauthConfig.RedirectURL = redirectURL

	// Generate PKCE verifier
	verifier := oauth2.GenerateVerifier()

	// Generate auth URL
	authURL := oauthConfig.AuthCodeURL("state",
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

	// Print URL to stderr
	fmt.Fprintln(errOut, "Open this URL in your browser:")
	fmt.Fprintln(errOut, authURL)

	// Start callback server
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code in callback", http.StatusBadRequest)
			errCh <- fmt.Errorf("no code in callback")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><h1>Authentication successful</h1><p>You may close this window.</p></body></html>")
		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for callback or timeout
	var code string
	select {
	case code = <-codeCh:
		// Got code
	case err := <-errCh:
		fmt.Fprintf(errOut, "error: %v\n", err)
		return exitcode.AuthError
	case <-time.After(oauthCallbackTimeout):
		fmt.Fprintln(errOut, "error: oauth callback timed out")
		return exitcode.AuthError
	case <-ctx.Done():
		fmt.Fprintln(errOut, "error: cancelled")
		return exitcode.AuthError
	}

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	// Exchange code for token
	exchangeCtx, cancelExchange := context.WithTimeout(ctx, tokenExchangeTimeout)
	defer cancelExchange()

	token, err := oauthConfig.Exchange(exchangeCtx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		fmt.Fprintf(errOut, "error: failed to exchange code for token: %v\n", err)
		return exitcode.AuthError
	}

	// Ensure config directory exists
	if err := cfg.EnsureDir(); err != nil {
		fmt.Fprintf(errOut, "error: failed to create config directory: %v\n", err)
		return exitcode.AuthError
	}

	// Save token
	if err := saveToken(cfg.TokenPath(), token); err != nil {
		fmt.Fprintf(errOut, "error: failed to save token: %v\n", err)
		return exitcode.AuthError
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}

// findAvailablePort tries to find an available port starting from oauthStartPort.
func findAvailablePort() (int, net.Listener, error) {
	for i := 0; i < oauthMaxPortAttempts; i++ {
		port := oauthStartPort + i
		addr := fmt.Sprintf("localhost:%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			return port, listener, nil
		}
	}
	return 0, nil, fmt.Errorf("no available port found")
}

// isTokenValid checks if a token file contains a valid token.
// Valid means: parseable and contains a non-empty refresh token.
func isTokenValid(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return false
	}
	return token.RefreshToken != ""
}

// saveToken saves an OAuth token to a file with mode 0600.
func saveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
