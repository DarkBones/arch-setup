package github

import (
	"os/exec"
	"testing"
)

type mockExecutor struct {
	stderrOutput string
}

func (m *mockExecutor) Run(cmd *exec.Cmd) error {
	if cmd.Stderr != nil {
		_, err := cmd.Stderr.Write([]byte(m.stderrOutput))
		if err != nil {
			panic("mockExecutor failed to write to stderr")
		}
	}
	return nil
}

func TestIsSshConnectionSuccessful_Success(t *testing.T) {
	// Arrange
	expectedOutput := "Hi test-user! You've successfully authenticated, but GitHub does not provide shell access."
	mockExec := &mockExecutor{
		stderrOutput: expectedOutput,
	}
	auth := newAuthenticator(mockExec)

	// Act
	isAuthenticated, username, output := auth.checkConnection()

	// Assert
	if !isAuthenticated {
		t.Error("expected isAuthenticated to be true, but got false")
	}

	expectedUsername := "test-user"
	if username != expectedUsername {
		t.Errorf("expected username to be %q, but got %q", expectedUsername, username)
	}

	if output != expectedOutput {
		t.Errorf("expected output to be %q, but got %q", expectedOutput, output)
	}
}

func TestIsSshConnectionSuccessful_Failure(t *testing.T) {
	// Arrange
	mockExec := &mockExecutor{
		stderrOutput: "git@github.com: Permission denied (publickey).",
	}
	auth := newAuthenticator(mockExec)

	// Act
	isAuthenticated, username, _ := auth.checkConnection()

	// Assert
	if isAuthenticated {
		t.Error("expected isAuthenticated to be false, but got true")
	}
	if username != "" {
		t.Errorf("expected username to be empty, but got %q", username)
	}
}

func TestParseSshOutput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		input             string
		wantAuthenticated bool
		wantUsername      string
	}{
		{
			name:              "Successful authentication with username",
			input:             "Hi DarkBones! You've successfully authenticated, but GitHub does not provide shell access.",
			wantAuthenticated: true,
			wantUsername:      "DarkBones",
		},
		{
			name:              "Failed authentication",
			input:             "git@github.com: Permission denied (publickey).",
			wantAuthenticated: false,
			wantUsername:      "",
		},
		{
			name:              "Empty input string",
			input:             "",
			wantAuthenticated: false,
			wantUsername:      "",
		},
		{
			name:              "Successful authentication with known_hosts warning",
			input:             "Warning: Permanently added 'github.com' (ED25519) to the list of known hosts.\nHi DarkBones! You've successfully authenticated, but GitHub does not provide shell access.",
			wantAuthenticated: true,
			wantUsername:      "DarkBones",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotAuthenticated, gotUsername := parseSshOutput(tc.input)

			if gotAuthenticated != tc.wantAuthenticated {
				t.Errorf("want authenticated %v, got %v", tc.wantAuthenticated, gotAuthenticated)
			}

			if gotUsername != tc.wantUsername {
				t.Errorf("want username %q, got %q", tc.wantUsername, gotUsername)
			}
		})
	}
}
