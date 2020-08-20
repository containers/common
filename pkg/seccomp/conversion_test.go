package seccomp

import (
	"testing"
)

func TestGoArchToSeccompArchSuccess(t *testing.T) {
	for goArch, seccompArch := range goArchToSeccompArchMap {
		res, err := GoArchToSeccompArch(goArch)
		if err != nil {
			t.Fatalf("expected nil, but got error: %v", err)
		}
		if seccompArch != res {
			t.Fatalf("expected %s, but got: %s", seccompArch, res)
		}
	}
}

func TestGoArchToSeccompArchFailure(t *testing.T) {
	res, err := GoArchToSeccompArch("wrong")
	if err == nil {
		t.Fatal("expected error, but got nil")
	}
	if res != "" {
		t.Fatalf("expected empty res, but got: %s", res)
	}
}
