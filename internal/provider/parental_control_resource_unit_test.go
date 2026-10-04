package provider

import (
	"testing"

	"github.com/jfgmesquita/terraform-provider-nextdns/internal/nextdns"
)

func TestWithoutRecreation(t *testing.T) {
	entries := []nextdns.ParentalControlEntry{
		{ID: "netflix", Active: true, Recreation: true},
		{ID: "twitch", Active: false, Recreation: false},
	}

	strict := withoutRecreation(entries)

	for _, e := range strict {
		if e.Recreation {
			t.Errorf("%s: recreation = true, want false", e.ID)
		}
	}
	if !strict[0].Active || strict[1].Active {
		t.Errorf("active flags changed: %+v", strict)
	}
	if !entries[0].Recreation {
		t.Error("the original entries were modified")
	}
}
