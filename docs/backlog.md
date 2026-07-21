# Backlog

This file records planned outcomes that have been accepted but are not yet implemented. Remove entries when they are delivered or no longer intended.

## Limit aggregate Harbor API concurrency

Add `--harbor-max-concurrent-requests` and a corresponding Helm value to cap the total number of in-flight Harbor API requests across all controllers in one operator process. The limit should protect constrained Harbor installations during request bursts without being multiplied independently per controller.

Before implementation, decide the backwards-compatible default and whether request-rate limiting is also needed; concurrency limits bound simultaneous slow requests but do not bound requests per second.

## Make strict documentation builds work from Git worktrees

`make build-docs-site` mounts only the working tree into its container. In a linked Git worktree, `.git` points to metadata outside that mount, so the MkDocs Git revision date plugin cannot inspect the repository and the strict build fails.

Update the canonical documentation build so `make build-docs-site`, `make docs-build`, and therefore `make check` work without manual Docker mount arguments in both normal clones and linked Git worktrees. Preserve revision-date metadata in both environments.
