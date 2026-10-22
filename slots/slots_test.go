package slots

import "testing"

func env(kv map[string]string) func(string) string { return func(k string) string { return kv[k] } }
func mem(gigs int64) func() int64                  { return func() int64 { return gigs * gib } }

func TestTheDefaultIsOneSlot(t *testing.T) {
	r := Resolve(env(nil), mem(16))
	if r.Requested != 1 || r.Effective != 1 || r.Reason != "" {
		t.Fatalf("%+v", r)
	}
}

func TestARequestIsCappedByTheHardwareAndSaysSo(t *testing.T) {
	r := Resolve(env(map[string]string{EnvRequested: "4"}), mem(16))
	if r.Requested != 4 || r.Max != 1 || r.Effective != 1 {
		t.Fatalf("a 16 GiB machine must be held to 1 slot: %+v", r)
	}
	if r.Reason == "" || !contains(r.Reason, "16 GiB") || !contains(r.Reason, "4 slots were asked for") {
		t.Fatalf("the reason must say what happened: %q", r.Reason)
	}
}

func TestBiggerMachinesAllowMore(t *testing.T) {
	for _, c := range []struct {
		gigs int64
		max  int
	}{{16, 1}, {24, 1}, {32, 2}, {48, 2}, {64, 4}, {96, 4}, {128, 8}, {512, 8}} {
		r := Resolve(env(map[string]string{EnvRequested: "8"}), mem(c.gigs))
		if r.Max != c.max || r.Effective != c.max {
			t.Fatalf("%d GiB: want max %d, got %+v", c.gigs, c.max, r)
		}
	}
}

func TestARequestBelowTheAllowanceIsKept(t *testing.T) {
	r := Resolve(env(map[string]string{EnvRequested: "2"}), mem(128))
	if r.Effective != 2 || r.Reason != "" {
		t.Fatalf("%+v", r)
	}
}

func TestTheOperatorCanRaiseTheAllowanceForAKnownMachine(t *testing.T) {
	r := Resolve(env(map[string]string{EnvRequested: "4", EnvMax: "4"}), mem(16))
	if r.Effective != 4 || r.Max != 4 {
		t.Fatalf("an explicit override must stand: %+v", r)
	}
	// ... but it never exceeds the request.
	r = Resolve(env(map[string]string{EnvRequested: "2", EnvMax: "8"}), mem(16))
	if r.Effective != 2 {
		t.Fatalf("the allowance is never above the request: %+v", r)
	}
}

func TestAnInvalidValueIsOneSlotAndSaysSo(t *testing.T) {
	for _, bad := range []string{"0", "-3", "two", "1.5"} {
		r := Resolve(env(map[string]string{EnvRequested: bad}), mem(128))
		if r.Effective != 1 || r.Reason == "" {
			t.Fatalf("%q: %+v", bad, r)
		}
	}
	// an invalid override is not trusted: the rule applies
	r := Resolve(env(map[string]string{EnvRequested: "4", EnvMax: "lots"}), mem(16))
	if r.Max != 1 || r.Effective != 1 {
		t.Fatalf("%+v", r)
	}
}

func TestUnknownMemoryIsTheSafeAnswer(t *testing.T) {
	r := Resolve(env(map[string]string{EnvRequested: "4"}), func() int64 { return 0 })
	if r.Effective != 1 || !contains(r.Reason, "could not be read") {
		t.Fatalf("%+v", r)
	}
}

func TestHostMemoryIsReadableHere(t *testing.T) {
	if n := HostMemoryBytes(); n <= 0 {
		t.Skip("this system's memory cannot be read here")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
