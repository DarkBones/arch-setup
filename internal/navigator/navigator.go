package navigator

type Navigator[T ~int] struct {
	History []T
}

func New[T ~int](initialPhase T) Navigator[T] {
	return Navigator[T]{
		History: []T{initialPhase},
	}
}

// Current returns the current phase
func (n *Navigator[T]) Current() T {
	return n.History[len(n.History)-1]
}

// Push adds a new phase to the navigation stack.
func (n *Navigator[T]) Push(p T) {
	n.History = append(n.History, p)
}

// Pop removes the last phase, navigating back. Returns true if successful.
func (n *Navigator[T]) Pop() bool {
	if len(n.History) <= 1 {
		return false
	}

	n.History = n.History[:len(n.History)-1]
	return true
}

// Reset clears the history and sets the current phase to the one provided.
func (n *Navigator[T]) Reset(p T) {
	n.History = []T{p}
}
