package navigator

import "testing"

func TestNavigator(t *testing.T) {
	t.Run("it should initialize with the correct phase", func(t *testing.T) {
		nav := New(100)
		if nav.Current() != 100 {
			t.Errorf("expected current phase to be 100, got %d", nav.Current())
		}
	})

	t.Run("Push should add a phase to the history", func(t *testing.T) {
		nav := New(0)
		nav.Push(1)
		if nav.Current() != 1 {
			t.Errorf("expected current phase to be 1, got %d", nav.Current())
		}
		if len(nav.History) != 2 {
			t.Errorf("expected history length to be 2, got %d", len(nav.History))
		}
	})

	t.Run("Pop should remove a phase and return true", func(t *testing.T) {
		nav := New(0)
		nav.Push(1)

		canPop := nav.Pop()

		if !canPop {
			t.Error("expected Pop to return true")
		}
		if nav.Current() != 0 {
			t.Errorf("expected current phase to be 0, got %d", nav.Current())
		}
	})

	t.Run("Pop should do nothing and return false if at the root", func(t *testing.T) {
		nav := New(0)

		canPop := nav.Pop()

		if canPop {
			t.Error("expected Pop to return false")
		}
		if nav.Current() != 0 {
			t.Errorf("expected current phase to be 0, got %d", nav.Current())
		}
	})
}
