package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

type mockChecker struct {
	shouldFail bool
}

func (m mockChecker) Check() error {
	if m.shouldFail {
		return errors.New("mock sudo failure")
	}
	return nil
}

type mockApp struct {
	shouldFail bool
}

func (m *mockApp) Run() error {
	if m.shouldFail {
		return errors.New("mock app failure")
	}

	return nil
}

func TestRun_PrivilegeCheck_Success(t *testing.T) {
	t.Parallel()

	// Arrange
	checker := mockChecker{shouldFail: false}
	app := &mockApp{} // Create an instance of the mock app.

	// Act: Pass the mock app as the third argument.
	err := run(nil, checker, app)

	// Assert
	if err != nil {
		t.Errorf("run() returned an error even though privilege check succeeded: %v", err)
	}
}

func TestRun_PrivilegeCheck_Failure(t *testing.T) {
	t.Parallel()

	// Arrange
	checker := mockChecker{shouldFail: true}
	app := &mockApp{} // Create an instance of the mock app.

	// Act: Pass the mock app as the third argument.
	err := run(nil, checker, app)

	// Assert
	if err == nil {
		t.Fatal("run() did not return an error even though privilege check failed")
	}

	expectedMsg := "could not obtain administrative privileges"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error message to contain %q, but got %q", expectedMsg, err.Error())
	}
}

func TestRun_AppFailure(t *testing.T) {
	t.Parallel()

	// Arrange
	checker := mockChecker{shouldFail: false}
	app := &mockApp{shouldFail: true}

	// Act
	err := run(nil, checker, app)

	// Assert
	if err == nil {
		t.Fatal("run() did not return an error even though the app failed")
	}

	expectedMsg := "application error"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected error message to contain %q, but got %q", expectedMsg, err.Error())
	}
}

func TestSetupLogging(t *testing.T) {
	t.Parallel()

	t.Run("it does nothing when debug is disabled", func(t *testing.T) {
		var creatorCalled bool
		mockCreator := func() (*os.File, error) {
			creatorCalled = true
			return nil, nil
		}

		logFile, err := setupLogging(false, mockCreator)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if logFile != nil {
			t.Error("expected a nil file handle")
		}
		if creatorCalled {
			t.Error("creator function was called but should not have been")
		}
	})

	t.Run("it calls the creator when debug is enabled", func(t *testing.T) {
		var creatorCalled bool
		// Create a mock that returns a predictable error
		mockCreator := func() (*os.File, error) {
			creatorCalled = true
			return nil, errors.New("mock creation failed")
		}

		logFile, err := setupLogging(true, mockCreator)

		if !creatorCalled {
			t.Error("creator function was not called but should have been")
		}
		if logFile != nil {
			t.Error("expected a nil file handle on error")
		}
		if err == nil {
			t.Error("expected an error but got nil")
		}
	})
}
