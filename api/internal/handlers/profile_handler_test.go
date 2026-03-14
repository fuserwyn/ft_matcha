package handlers

import (
	"reflect"
	"testing"
)

func TestNormalizeSexualPreference(t *testing.T) {
	t.Run("nil stays nil", func(t *testing.T) {
		if got := normalizeSexualPreference(nil); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("empty becomes male female", func(t *testing.T) {
		values := []string{}
		want := []string{"male", "female"}
		got := normalizeSexualPreference(&values)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})

	t.Run("provided values preserved", func(t *testing.T) {
		values := []string{"female"}
		want := []string{"female"}
		got := normalizeSexualPreference(&values)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})
}
