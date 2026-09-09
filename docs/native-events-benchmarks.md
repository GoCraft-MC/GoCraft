# Native event measurements

Local development measurements, not a production latency guarantee. Go results
were collected September 8, 2026; JVM results September 9. Windows/amd64,
AMD Ryzen 5 7535HS (12 logical processors), Go 1.26.0, Oracle Java 25.0.1.
Background machine load was not controlled. Each batch ran 20 measured times;
these are arithmetic means, not p95/p99. Benchmarks ran without race instrumentation.
See [commands and methodology](native-events-validation.md#performance-methodology).

## Cold and first events

| Measurement | Go child | JVM child |
| --- | ---: | ---: |
| Process startup and plugin load | 432.9435 ms | 458.1016 ms |
| First event without shaped warm-up | 1.0775 ms | 41.4241 ms |

A separate fresh-JVM functional test loaded in 588.8658 ms, ran handler-free
shaped warm-up, and its first real handler dispatch took 4.2807 ms. It passed
using the existing 20 ms first-event grace; subsequent cancellation also passed.
Warm-up therefore reduces runtime initialization but does not eliminate all
plugin-owned first-handler work. The benchmark's first call after warm-up
reported `0s` at the local timer's granularity, not zero execution cost; that
benchmark had already executed its cold handler. Do not use it as evidence that
a fresh plugin's first real handler is free or always below 2 ms.

## Warmed batches

The host-only path uses a trivial in-process verdict provider, excluding IPC,
serialization and SDK execution. Both child-process fixtures rewrite a string.
The end-to-end measurements include that tiny handler; arbitrary plugin CPU
cost is not separately profiled. Differences between noisy means are not exact
component-cost measurements.

| Events per batch | Host-only batch | Go IPC batch | JVM IPC batch |
| ---: | ---: | ---: | ---: |
| 1 | 0.005510 ms | 0.151670 ms | 0.749920 ms |
| 100 | 0.226325 ms | 12.792795 ms | 23.315220 ms |
| 500 | 1.244800 ms | 65.102480 ms | 59.418525 ms |
| 600 | 1.635905 ms | 92.591565 ms | 104.577085 ms |
| 1000 | 2.218260 ms | 172.094570 ms | 172.759445 ms |

| Events per batch | Go mean/event | Go calls >2 ms | JVM mean/event | JVM calls >2 ms |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 151.570 us | 0% | 749.795 us | 0% |
| 100 | 127.927 us | 0% | 233.151 us | 0.3000% |
| 500 | 130.205 us | 0% | 118.837 us | 0.1000% |
| 600 | 154.319 us | 0.02500% | 174.295 us | 1.258% |
| 1000 | 172.094 us | 0.03500% | 172.759 us | 0.9650% |

The measurement-only deadline is one second so slow samples are retained rather
than losing returned mutations or disabling the fixture. Production remains at
the existing approximately 2 ms shared budget, with its unchanged failure policy
and first-event grace. Overrun percentages count complete local round trips;
they are not measurements of production timeout outcomes. No claim is made that
1000 IPC events fit a 50 ms server tick or share a single 2 ms allowance.

## Changes motivated by measurements

Plugin health recording rescanned its minute of history for every verdict,
causing quadratic growth in sustained batches. Rolling duration/failure counters
now preserve the same pruning window and disable threshold in amortized O(1).
The host-only 1000-event batch measured 66.6308 ms before that fix and 2.218260 ms
after it on this machine; this is a development observation, not a controlled
cross-version performance study.

JVM warm-up now also encodes a synthetic scalar mutation verdict locally to
initialize protobuf mutation classes. It does not invoke plugin handlers,
change the warmed event, or return synthetic effects/mutations. Tests assert
those properties. Traqueur's shaped warm-up and cold grace were retained.

The warmed means are below 2 ms, but observed outliers remain, particularly on
Java. Linux measurements, sustained mixed-event workloads, percentiles and
profiling of GC/scheduling tails remain necessary before a stronger latency claim.
