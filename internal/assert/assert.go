package assert

import (
	"fmt"
	"reflect"
	"runtime/debug"
)

// Panics with the given msg if the given condition is false
func True(condition bool, msg string) {
	if !condition {
		logFailure(msg)
	}
}

// Panics with the given msg if the given condition is true
func False(condition bool, msg string) {
	if condition {
		logFailure(msg)
	}
}

// Panics with the given msg if the given condition is not nil
func NotNil(item any, msg string) {
	if item == nil || reflect.ValueOf(item).Kind() == reflect.Ptr && reflect.ValueOf(item).IsNil() {
		logFailure(msg)
	}
}

// Panics with the given msg if the given condition is nil
func Nil(item any, msg string) {
	if item != nil {
		logFailure(msg)
	}
}

// Panics with the given msg + error message + stack trace if the given error
// is not nil
func NoError(err error, msg string) {
	if err != nil {
		logFailure(fmt.Sprintf("%s: %v (type: %T)", msg, err, err))
	}
}

// Panics with the given msg
func Fail(msg string) {
	logFailure(msg)
}

func logFailure(msg string) {
	errMsg := fmt.Sprintf("Assertion failed: %s\n", msg)
	errMsg += fmt.Sprintf("%s", string(debug.Stack()))
	panic(errMsg)
}
