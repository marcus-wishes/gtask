package commands_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"gtask/internal/commands"
	"gtask/internal/config"
	"gtask/internal/exitcode"
)

// TestLoginCommand_NoOAuthClient verifies login fails without oauth_client.json
func TestLoginCommand_NoOAuthClient(t *testing.T) {
	cmd := &commands.LoginCmd{}

	var outBuf, errBuf bytes.Buffer
	cfg := &config.Config{
		Dir:   t.TempDir(),
		Quiet: false,
	}

	ctx := context.Background()
	code := cmd.Run(ctx, cfg, nil, nil, &outBuf, &errBuf)

	if code != exitcode.AuthError {
		t.Errorf("expected exit code %d, got %d", exitcode.AuthError, code)
	}
	if outBuf.String() != "" {
		t.Errorf("expected no stdout, got %q", outBuf.String())
	}
	if errBuf.String() == "" {
		t.Error("expected error message about missing oauth_client.json")
	}
}

// TestLoginCommand_InvalidToken verifies login proceeds when token is invalid/corrupt
func TestLoginCommand_InvalidToken(t *testing.T) {
	cmd := &commands.LoginCmd{}

	tmpDir := t.TempDir()

	// Create oauth_client.json
	oauthClient := `{"installed":{"client_id":"test","client_secret":"test","redirect_uris":["http://localhost"]}}`
	err := os.WriteFile(filepath.Join(tmpDir, "oauth_client.json"), []byte(oauthClient), 0600)
	if err != nil {
		t.Fatalf("failed to write oauth_client.json: %v", err)
	}

	// Create invalid token.json (no refresh token)
	invalidToken := `{"access_token":"expired","token_type":"Bearer"}`
	err = os.WriteFile(filepath.Join(tmpDir, "token.json"), []byte(invalidToken), 0600)
	if err != nil {
		t.Fatalf("failed to write token.json: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	cfg := &config.Config{
		Dir:   tmpDir,
		Quiet: false,
	}

	// Create a context that cancels immediately to prevent waiting for OAuth callback
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_ = cmd.Run(ctx, cfg, nil, nil, &outBuf, &errBuf)

	// Should try to proceed with login (context cancelled, so it will error)
	// The important thing is it didn't say "already logged in"
	if outBuf.String() == "already logged in\n" {
		t.Error("should not say 'already logged in' with invalid token")
	}
}

// TestLoginCommand_NoRefreshToken verifies login proceeds when token has no refresh token
func TestLoginCommand_NoRefreshToken(t *testing.T) {
	cmd := &commands.LoginCmd{}

	tmpDir := t.TempDir()

	// Create oauth_client.json
	oauthClient := `{"installed":{"client_id":"test","client_secret":"test","redirect_uris":["http://localhost"]}}`
	err := os.WriteFile(filepath.Join(tmpDir, "oauth_client.json"), []byte(oauthClient), 0600)
	if err != nil {
		t.Fatalf("failed to write oauth_client.json: %v", err)
	}

	// Create token.json without refresh_token
	tokenWithoutRefresh := `{"access_token":"test","token_type":"Bearer","expiry":"2020-01-01T00:00:00Z"}`
	err = os.WriteFile(filepath.Join(tmpDir, "token.json"), []byte(tokenWithoutRefresh), 0600)
	if err != nil {
		t.Fatalf("failed to write token.json: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	cfg := &config.Config{
		Dir:   tmpDir,
		Quiet: false,
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code := cmd.Run(ctx, cfg, nil, nil, &outBuf, &errBuf)

	// Should try to proceed with login (not "already logged in")
	if outBuf.String() == "already logged in\n" {
		t.Error("should not say 'already logged in' with token missing refresh_token")
	}
	_ = code // We don't care about the exact exit code, just that it tried to re-login
}

// TestLogoutCommand_OnlyRemovesToken verifies logout only removes token.json
func TestLogoutCommand_OnlyRemovesToken(t *testing.T) {
	cmd := &commands.LogoutCmd{}

	tmpDir := t.TempDir()

	// Create oauth_client.json
	oauthClient := `{"installed":{"client_id":"test","client_secret":"test"}}`
	oauthPath := filepath.Join(tmpDir, "oauth_client.json")
	err := os.WriteFile(oauthPath, []byte(oauthClient), 0600)
	if err != nil {
		t.Fatalf("failed to write oauth_client.json: %v", err)
	}

	// Create token.json
	tokenPath := filepath.Join(tmpDir, "token.json")
	err = os.WriteFile(tokenPath, []byte(`{"access_token":"test","refresh_token":"test"}`), 0600)
	if err != nil {
		t.Fatalf("failed to write token.json: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	cfg := &config.Config{
		Dir:   tmpDir,
		Quiet: false,
	}

	ctx := context.Background()
	code := cmd.Run(ctx, cfg, nil, nil, &outBuf, &errBuf)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if errBuf.String() != "" {
		t.Errorf("expected no stderr, got %q", errBuf.String())
	}
	if outBuf.String() != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", outBuf.String())
	}

	// Verify token.json was deleted
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Error("token.json should have been deleted")
	}

	// Verify oauth_client.json still exists
	if _, err := os.Stat(oauthPath); err != nil {
		t.Error("oauth_client.json should NOT have been deleted")
	}
}

// TestLogoutCommand_NotLoggedIn verifies logout handles not being logged in
func TestLogoutCommand_NotLoggedIn(t *testing.T) {
	cmd := &commands.LogoutCmd{}

	var outBuf, errBuf bytes.Buffer
	cfg := &config.Config{
		Dir:   t.TempDir(),
		Quiet: false,
	}

	ctx := context.Background()
	code := cmd.Run(ctx, cfg, nil, nil, &outBuf, &errBuf)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if errBuf.String() != "" {
		t.Errorf("expected no stderr, got %q", errBuf.String())
	}
	if outBuf.String() != "not logged in\n" {
		t.Errorf("expected 'not logged in\\n', got %q", outBuf.String())
	}
}

// TestLogoutCommand_NotLoggedInQuiet verifies logout is quiet when not logged in
func TestLogoutCommand_NotLoggedInQuiet(t *testing.T) {
	cmd := &commands.LogoutCmd{}

	var outBuf, errBuf bytes.Buffer
	cfg := &config.Config{
		Dir:   t.TempDir(),
		Quiet: true,
	}

	ctx := context.Background()
	code := cmd.Run(ctx, cfg, nil, nil, &outBuf, &errBuf)

	if code != exitcode.Success {
		t.Errorf("expected exit code %d, got %d", exitcode.Success, code)
	}
	if errBuf.String() != "" {
		t.Errorf("expected no stderr, got %q", errBuf.String())
	}
	if outBuf.String() != "" {
		t.Errorf("expected no stdout in quiet mode, got %q", outBuf.String())
	}
}
