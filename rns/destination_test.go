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

func TestDestinationSetStampCostParity(t *testing.T) {
	d := &Destination{}

	d.SetStampCost(nil)
	if d.StampCost != nil {
		t.Fatalf("stamp_cost=%#v, want nil", d.StampCost)
	}

	d.SetStampCost(0)
	if d.StampCost != nil {
		t.Fatalf("stamp_cost=%#v after 0, want nil", d.StampCost)
	}

	d.SetStampCost(-1)
	if d.StampCost != nil {
		t.Fatalf("stamp_cost=%#v after -1, want nil", d.StampCost)
	}

	d.SetStampCost(1)
	if cost, ok := d.StampCost.(int); !ok || cost != 1 {
		t.Fatalf("stamp_cost=%#v after 1, want int(1)", d.StampCost)
	}

	d.SetStampCost(254)
	if cost, ok := d.StampCost.(int); !ok || cost != 254 {
		t.Fatalf("stamp_cost=%#v after 254, want int(254)", d.StampCost)
	}

	d.SetStampCost(255)
	if flag, ok := d.StampCost.(bool); !ok || flag {
		t.Fatalf("stamp_cost=%#v after 255, want false", d.StampCost)
	}

	d.SetStampCost(uint8(42))
	if cost, ok := d.StampCost.(int); !ok || cost != 42 {
		t.Fatalf("stamp_cost=%#v after uint8(42), want int(42)", d.StampCost)
	}

	d.SetStampCost(false)
	if flag, ok := d.StampCost.(bool); !ok || flag {
		t.Fatalf("stamp_cost=%#v after false, want false", d.StampCost)
	}
}

func TestDestinationSetStampCostRejectsTrue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for true stamp cost")
		}
	}()

	(&Destination{}).SetStampCost(true)
}
