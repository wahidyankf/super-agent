package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/wahidyankf/super-agent/apps/retriever-cli/tests/support/outlinestub"
	"testing"
)

func TestRunRefusesToStartWithoutACAFile(t *testing.T) {
	err := run(context.Background(), []string{}, &bytes.Buffer{})

	if !errors.Is(err, errNoCAFile) {
		t.Fatalf("run without -ca-file returned %v, want errNoCAFile", err)
	}
}

func TestCollectionFlagReadsIDAndNameInListingOrder(t *testing.T) {
	var flags collectionFlag

	for _, value := range []string{"c1=Engineering", "c2=R&D = Drafts"} {
		if err := flags.Set(value); err != nil {
			t.Fatalf("Set(%q): %v", value, err)
		}
	}

	want := []outlinestub.Collection{{ID: "c1", Name: "Engineering"}, {ID: "c2", Name: "R&D = Drafts"}}
	if len(flags) != len(want) || flags[0] != want[0] || flags[1] != want[1] {
		t.Fatalf("collections = %v, want %v", flags, want)
	}
}

func TestCollectionFlagRefusesAValueWithNoIDOrName(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "no separator", value: "no-equals-sign", want: `-collection wants id=name, got "no-equals-sign"`},
		{name: "empty id", value: "=Engineering", want: `-collection wants id=name, got "=Engineering"`},
		{name: "empty name", value: "c1=", want: `-collection wants id=name, got "c1="`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var flags collectionFlag
			err := flags.Set(tc.value)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("Set(%q) returned %v, want %q", tc.value, err, tc.want)
			}
			if len(flags) != 0 {
				t.Fatalf("Set(%q) stored %v after refusing it", tc.value, flags)
			}
		})
	}
}
