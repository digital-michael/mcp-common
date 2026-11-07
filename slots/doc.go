// Package slots decides how many requests the model engine serves at once, the one number the
// agent queue and the chat's own scheduler both depend on.
//
// The setting is DATUM_MODEL_SLOTS (default 1): how many slots the operator asks for, meant to
// equal the engine's own parallelism (OLLAMA_NUM_PARALLEL). It is governed by the hardware: each
// extra slot holds its own context in memory, so a machine that cannot afford another model-sized
// working set must not be asked to run one, whatever the setting says. The effective count is the
// smaller of what was asked and what the hardware allows. The allowance is a rule of thumb by
// total memory (Tiers), which the operator can override for a machine known to be more capable
// with DATUM_MODEL_SLOTS_MAX; it is never above the request.
//
// Measured on the 16 GiB Apple-silicon laptop this project is developed on (Ollama's load log,
// 2026-10-22): the 9B model at a 32,768-token window holds 9.1 GiB, the 4B 6.5 GiB, of the
// 11.8 GiB the engine may use; a second slot would need roughly the context working memory again
// (about 2.9 GiB per 32k), which does not fit beside a model of that size. So a laptop of that
// class is one slot.
//
// # Goals
//
//   - One answer to "how many model requests at once", shared by every component that queues them.
//   - Never ask the machine for more parallelism than its memory allows, and say why when capped.
//
// # Non-goals
//
//   - Choosing or loading models, or reading the engine's own configuration.
//   - Scheduling. Callers queue; this package only says how wide the queue may be.
//
// # Boundaries
//
// Owns: the two settings, the memory tiers and Resolve.
// Depends on: the standard library, and sysctl or /proc to read total memory.
// Used by: mcp-agent.
package slots
