package assert

import (
	"fmt"
	"strings"
	"testing"
)

func TestTrueWithTrueCondition(t *testing.T) {
	True(true, "This should not panic")
}

func TestTrueWithFalseCondition(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but function did not panic")
		}
	}()

	True(false, "This should panic")
}

func TestFalseWithTrueCondition(t *testing.T) {
	False(false, "This should not panic")
}

func TestFalseWithFalseCondition(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but function did not panic")
		}
	}()

	False(true, "This should panic")
}

func TestNotNilWithNonNilValue(t *testing.T) {
	NotNil("something", "This should not panic")
}

func TestNotNilWithNilValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but function did not panic")
		}
	}()

	NotNil(nil, "This should panic")
}

func TestNilWithNilValue(t *testing.T) {
	Nil(nil, "This should not panic")
}

func TestNilWithNonNilValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but function did not panic")
		}
	}()

	Nil("something", "This should panic")
}

func TestNoErrorWithNilError(t *testing.T) {
	NoError(nil, "This should not panic")
}

func TestNoErrorWithError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but function did not panic")
		} else if msg := r.(string); !strings.Contains(msg, "an error occurred (type: *errors.errorString)\n") {
			t.Fatalf("Unexpected panic message: got %v", msg)
		}
	}()

	NoError(fmt.Errorf("an error occurred"), "This should panic")
}

func TestFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic, but function did not panic")
		} else if msg := r.(string); !strings.Contains(msg, "This should always panic") {
			t.Fatalf("Unexpected panic message: got %v", msg)
		}
	}()

	Fail("This should always panic")
}
