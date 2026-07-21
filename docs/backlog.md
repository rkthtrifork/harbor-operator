# Backlog

This file records planned outcomes that have been accepted but are not yet implemented. Remove entries when they are delivered or no longer intended.

## Limit aggregate Harbor API concurrency

Add `--harbor-max-concurrent-requests` and a corresponding Helm value to cap the total number of in-flight Harbor API requests across all controllers in one operator process. The limit should protect constrained Harbor installations during request bursts without being multiplied independently per controller.

Before implementation, decide the backwards-compatible default and whether request-rate limiting is also needed; concurrency limits bound simultaneous slow requests but do not bound requests per second.
