package slots

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

const (
	// EnvRequested is the number of slots asked for. It should equal the engine's own parallelism.
	EnvRequested = "DATUM_MODEL_SLOTS"
	// EnvMax overrides the hardware allowance for a machine the operator knows to be more capable.
	EnvMax = "DATUM_MODEL_SLOTS_MAX"
)

const gib = int64(1) << 30

// Tier is one row of the hardware rule: up to this much memory, at most this many slots.
type Tier struct {
	UpToBytes int64
	Max       int
}

// Tiers is the hardware governance, by total memory. A row is a rule of thumb, not a measurement of every model: change a row (or set
// DATUM_MODEL_SLOTS_MAX) when a machine class proves otherwise.
var Tiers = []Tier{
	{UpToBytes: 24 * gib, Max: 1},
	{UpToBytes: 48 * gib, Max: 2},
	{UpToBytes: 96 * gib, Max: 4},
	{UpToBytes: math.MaxInt64, Max: 8},
}

// Result is what was asked, what the hardware allows, and what will be used.
type Result struct {
	Requested int
	Max       int
	Effective int
	// Reason says, in a sentence, anything the operator should know: a request that was capped, an invalid value, memory not known.
	Reason string
}

// Requested reads the setting. An absent value is 1; an invalid one (not a whole number of at least 1) is 1, and says so.
func Requested(getenv func(string) string) (int, string) {
	raw := strings.TrimSpace(lookup(getenv, EnvRequested))
	if raw == "" {
		return 1, ""
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1, fmt.Sprintf("%s=%q is not a whole number of at least 1, so 1 slot is used", EnvRequested, raw)
	}
	return n, ""
}

// MaxFor is the hardware allowance for a machine with this much memory (0 when it could not be read: one slot, the safe answer).
func MaxFor(totalBytes int64, getenv func(string) string) (int, string) {
	if raw := strings.TrimSpace(lookup(getenv, EnvMax)); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			return n, fmt.Sprintf("the hardware allowance is %d, set by %s", n, EnvMax)
		}
		// an invalid override falls through to the rule rather than being trusted
	}
	if totalBytes <= 0 {
		return 1, "this machine's memory could not be read, so 1 slot is allowed"
	}
	for _, t := range Tiers {
		if totalBytes <= t.UpToBytes {
			return t.Max, fmt.Sprintf("this machine has %.0f GiB of memory, which allows %d slot(s)", float64(totalBytes)/float64(gib), t.Max)
		}
	}
	return 1, ""
}

// Resolve combines the request with the hardware allowance. host reads total memory in bytes (HostMemoryBytes, or a fake in tests).
func Resolve(getenv func(string) string, host func() int64) Result {
	req, reqNote := Requested(getenv)
	max, maxNote := MaxFor(host(), getenv)
	r := Result{Requested: req, Max: max, Effective: req}
	if req > max {
		r.Effective = max
		r.Reason = fmt.Sprintf("%d slots were asked for but %s, so %d is used", req, maxNote, max)
	} else if reqNote != "" {
		r.Reason = reqNote
	}
	return r
}

// HostMemoryBytes reads this machine's total memory, or 0 when it cannot (an unsupported system, a failed read).
func HostMemoryBytes() int64 {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0
		}
		n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return 0
		}
		return n
	case "linux":
		raw, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				f := strings.Fields(line)
				if len(f) >= 2 {
					if kb, err := strconv.ParseInt(f[1], 10, 64); err == nil {
						return kb * 1024
					}
				}
			}
		}
	}
	return 0
}

// lookup reads a DATUM_ variable; an unset or blank one is empty. (The CORTEX_ names used until 2026-11-03 are no longer read.)
func lookup(getenv func(string) string, name string) string {
	if v := getenv(name); strings.TrimSpace(v) != "" {
		return v
	}
	return ""
}
