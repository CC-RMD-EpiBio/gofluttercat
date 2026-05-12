package biascorrection

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestBCMApplyMonotone(t *testing.T) {
	bcm := &BCM{
		XThresholds: []float64{-2.0, -1.0, 0.0, 1.0, 2.0},
		YThresholds: []float64{-1.5, -0.7, 0.0, 0.7, 1.5},
	}
	if err := bcm.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// Exact threshold matches.
	for i, x := range bcm.XThresholds {
		got := bcm.Apply(x)
		want := bcm.YThresholds[i]
		if math.Abs(got-want) > 1e-12 {
			t.Errorf("Apply(%g) = %g, want %g", x, got, want)
		}
	}

	// Linear interpolation between two adjacent points.
	got := bcm.Apply(-0.5)
	want := -0.35 // halfway between (-1, -0.7) and (0, 0.0)
	if math.Abs(got-want) > 1e-12 {
		t.Errorf("Apply(-0.5) = %g, want %g", got, want)
	}

	// Clipping below.
	if got := bcm.Apply(-99.0); got != bcm.YThresholds[0] {
		t.Errorf("Apply(-99) = %g, want %g (clipped low)", got, bcm.YThresholds[0])
	}
	// Clipping above.
	if got := bcm.Apply(99.0); got != bcm.YThresholds[len(bcm.YThresholds)-1] {
		t.Errorf("Apply(99) = %g, want %g (clipped high)", got, bcm.YThresholds[len(bcm.YThresholds)-1])
	}
}

func TestBCMValidate(t *testing.T) {
	cases := []struct {
		name string
		bcm  *BCM
		want bool // want error
	}{
		{"length mismatch", &BCM{XThresholds: []float64{0, 1}, YThresholds: []float64{0}}, true},
		{"too few", &BCM{XThresholds: []float64{0}, YThresholds: []float64{0}}, true},
		{"x not sorted", &BCM{XThresholds: []float64{1, 0}, YThresholds: []float64{0, 1}}, true},
		{"y not monotone", &BCM{XThresholds: []float64{0, 1}, YThresholds: []float64{1, 0}}, true},
		{"ok", &BCM{XThresholds: []float64{0, 1}, YThresholds: []float64{0, 1}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.bcm.Validate()
			if (err != nil) != c.want {
				t.Errorf("Validate err=%v, want_err=%v", err, c.want)
			}
		})
	}
}

func TestSetForFallback(t *testing.T) {
	set := &Set{
		Scale: "test",
		Maps: map[int]*BCM{
			5:  {SubsetSize: 5, XThresholds: []float64{-1, 1}, YThresholds: []float64{0, 0.5}},
			10: {SubsetSize: 10, XThresholds: []float64{-1, 1}, YThresholds: []float64{-0.5, 0.5}},
		},
	}

	if set.For(5) == nil || set.For(5).SubsetSize != 5 {
		t.Errorf("For(5) should return SubsetSize=5")
	}
	if set.For(10) == nil || set.For(10).SubsetSize != 10 {
		t.Errorf("For(10) should return SubsetSize=10")
	}
	// Closer to 5 than to 10 (|7-5|=2 vs |7-10|=3).
	if got := set.For(7); got.SubsetSize != 5 {
		t.Errorf("For(7) should fall back to 5, got %d", got.SubsetSize)
	}
	// Closer to 10.
	if got := set.For(8); got.SubsetSize != 10 {
		t.Errorf("For(8) should fall back to 10, got %d", got.SubsetSize)
	}

	empty := &Set{Maps: map[int]*BCM{}}
	if empty.For(5) != nil {
		t.Errorf("empty Set.For should return nil")
	}
}

func TestLoadSetRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bcm.json")
	js := `{
		"scale": "scs",
		"maps": {
			"5":  {"x_thresholds": [-2, 0, 2], "y_thresholds": [-1, 0, 1]},
			"10": {"x_thresholds": [-2, 0, 2], "y_thresholds": [-0.5, 0, 0.5]}
		}
	}`
	if err := os.WriteFile(path, []byte(js), 0644); err != nil {
		t.Fatal(err)
	}
	set, err := LoadSet(path)
	if err != nil {
		t.Fatalf("LoadSet: %v", err)
	}
	if set.Scale != "scs" {
		t.Errorf("Scale = %q, want scs", set.Scale)
	}
	if len(set.Maps) != 2 {
		t.Errorf("len(Maps) = %d, want 2", len(set.Maps))
	}
	bcm5 := set.For(5)
	if bcm5.Scale != "scs" {
		t.Errorf("BCM[5].Scale = %q, want scs (inherited)", bcm5.Scale)
	}
	if got := bcm5.Apply(0.0); math.Abs(got) > 1e-12 {
		t.Errorf("BCM[5].Apply(0) = %g, want 0", got)
	}
	bcm10 := set.For(10)
	if got := bcm10.Apply(2.0); math.Abs(got-0.5) > 1e-12 {
		t.Errorf("BCM[10].Apply(2) = %g, want 0.5", got)
	}
}

func TestLoadSetInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSet(path); err == nil {
		t.Error("LoadSet should fail on invalid JSON")
	}
}
