package assert

import "testing"

func Equal[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Errorf(`expected %#v but got %#v`, expected, actual)
	}
}

func AnyEqual(t *testing.T, actual, expected any) {
	t.Helper()
	if actual != expected {
		t.Errorf(`expected %#v but got %#v`, expected, actual)
	}
}

func NoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf(`expected no error but got %v`, err)
	}
}
