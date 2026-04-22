package rns

import "testing"

func TestDestinationExpandNameRejectsDots(t *testing.T) {
	if _, err := (&Destination{}).ExpandName(nil, "app.withdot"); err == nil {
		t.Fatalf("expected error for dot in app name")
	}
	if _, err := (&Destination{}).ExpandName(nil, "app", "aspect.withdot"); err == nil {
		t.Fatalf("expected error for dot in aspect")
	}
}

func TestDestinationAppAndAspectsFromName(t *testing.T) {
	app, aspects := (&Destination{}).AppAndAspectsFromName("rnstransport.remote.management")
	if app != "rnstransport" {
		t.Fatalf("unexpected app %q", app)
	}
	if len(aspects) != 2 || aspects[0] != "remote" || aspects[1] != "management" {
		t.Fatalf("unexpected aspects %#v", aspects)
	}
}
