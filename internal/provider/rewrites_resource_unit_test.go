package provider

import (
	"reflect"
	"testing"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

func TestDiffRewrites(t *testing.T) {
	current := []nextdns.Rewrite{
		{ID: "keep", Name: "router.home", Content: "192.168.1.1"},
		{ID: "remove", Name: "old.home", Content: "192.168.1.9"},
		{ID: "dup", Name: "router.home", Content: "192.168.1.1"},
		{ID: "v6-keep", Name: "v6.home", Content: "fd00::1"},
	}
	desired := []rewriteKey{
		{"router.home", "192.168.1.1"},
		{"v6.home", "fd00::1"},
		{"v6.home", "fd00::2"}, // same domain, second answer
		{"www.example.com", "example.org"},
	}

	toDelete, toCreate := diffRewrites(current, desired)

	if want := []string{"remove", "dup"}; !reflect.DeepEqual(toDelete, want) {
		t.Errorf("toDelete = %v, want %v", toDelete, want)
	}
	if want := []rewriteKey{{"v6.home", "fd00::2"}, {"www.example.com", "example.org"}}; !reflect.DeepEqual(toCreate, want) {
		t.Errorf("toCreate = %v, want %v", toCreate, want)
	}
}

func TestDiffRewritesNothingToDo(t *testing.T) {
	current := []nextdns.Rewrite{{ID: "a", Name: "router.home", Content: "192.168.1.1"}}
	desired := []rewriteKey{{"router.home", "192.168.1.1"}}

	toDelete, toCreate := diffRewrites(current, desired)
	if len(toDelete) != 0 || len(toCreate) != 0 {
		t.Errorf("toDelete = %v, toCreate = %v, want nothing", toDelete, toCreate)
	}
}
