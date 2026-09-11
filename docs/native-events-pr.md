# Suggested pull request

Title: `Complete typed Go event round trips and fundamental gameplay hooks`

## Summary

- Combine `feat/custom-events` and the latest `missing-items` on
  `feat/go-events-api`, preserving both gameplay parity and event architecture.
- Extend the common ABI schema/generator with explicit native mutable fields;
  generate typed Go callbacks and compatible Java setters/verdicts.
- Add 12 fundamental native events, with cancellation, validated mutations,
  duplicate prevention and client reconciliation at current gameplay hooks.
- Cover real Go/JVM processes, packet/intent paths, reload, generated mappings,
  examples and warm-up; fix quadratic health-history accounting.

## Validation

- GoCraft, Go SDK and ABI test suites pass; focused runtime/event race tests pass.
- JVM Gradle build/tests pass; Go and Java example bundles build.
- Regenerating the shared schema leaves no generated code differences.
- Real IPC tests verify returned chat cancellation/mutation on both runtimes.
- Cold/first/warm and 100/500/600/1000-event measurements are recorded in
  [the benchmark report](native-events-benchmarks.md), including >2 ms outliers.

## Compatibility and limits

Requires the matching `feat/go-events-api` companion branches in gocraft-abi,
gocraft-api-go, gocraft-jvm and gocraft-plugin-examples. Go dependencies pin the
published feature commits; the Java example currently uses Maven local artifacts.
The typed Go OnPlayerJoin callback now omits EventControl. Existing custom-event
architecture, transport, priorities and first-event grace remain in place.

No complete Minecraft bot login/action test was available. Structured killer
data, specialised gameplay hooks and sustained Linux/tail-latency profiling
remain follow-up work; see [the implementation report](native-events-report.md).

This is a suggested PR body, not an automatically created pull request.
