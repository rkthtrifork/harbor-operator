#!/usr/bin/env python3
"""Create automated dependency-only patch releases for release branches."""

from __future__ import annotations

import json
import os
import re
import shlex
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

from required_checks import DEFAULT_REQUIRED_CHECKS, wait_for_required_checks as wait_for_checks


DRY_RUN = os.environ.get("DRY_RUN", "false") == "true"
RELEASE_BRANCH = os.environ.get("RELEASE_BRANCH", "")
SUPPORTED_RELEASE_BRANCH_COUNT = int(os.environ.get("SUPPORTED_RELEASE_BRANCH_COUNT", "3"))
GITHUB_OUTPUT = os.environ.get("GITHUB_OUTPUT", "")

REQUIRED_CHECKS = DEFAULT_REQUIRED_CHECKS
AUTO_RELEASE_TRIGGER_PATHS = {
    "go.mod",
    "go.sum",
    "Dockerfile",
}
OPERATOR_RUNTIME_PATHS = ("go.mod", "go.sum", "Dockerfile", "api", "cmd", "internal")
CHART_PATH = "charts/harbor-operator"
CHART_YAML_PATH = f"{CHART_PATH}/Chart.yaml"
CHART_YAML = Path("charts/harbor-operator/Chart.yaml")
STABLE_VERSION_RE = re.compile(r"^[0-9]+\.[0-9]+\.[0-9]+$")
BRANCH_RE = re.compile(r"^release/v?[0-9]+\.[0-9]+$")

operator_releases: list[dict[str, str]] = []
chart_releases: list[dict[str, str]] = []


def log(message: str) -> None:
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    print(f"[{now}] {message}", flush=True)


def run(args: list[str], *, check: bool = True) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(args, check=check, text=True, capture_output=True)
    except subprocess.CalledProcessError as err:
        print(f"Command failed with exit code {err.returncode}: {shlex.join(args)}", file=sys.stderr)
        if err.stdout:
            print("--- stdout ---", file=sys.stderr)
            print(err.stdout.rstrip(), file=sys.stderr)
        if err.stderr:
            print("--- stderr ---", file=sys.stderr)
            print(err.stderr.rstrip(), file=sys.stderr)
        raise


def run_out(args: list[str], *, check: bool = True) -> str:
    return run(args, check=check).stdout.strip()


def semver_key(version: str) -> tuple[int, int, int]:
    major, minor, patch = version.split(".")
    return int(major), int(minor), int(patch)


def increment_patch(version: str) -> str:
    major, minor, patch = semver_key(version)
    return f"{major}.{minor}.{patch + 1}"


def tag_version(tag: str, prefix: str) -> str:
    return tag.removeprefix(prefix)


def stable_tags_merged(merged_ref: str, prefix: str, line: str | None = None) -> list[str]:
    pattern = f"{prefix}{line}.*" if line else f"{prefix}*"
    result = run_out(["git", "tag", "--merged", merged_ref, "--list", pattern], check=False)
    tags = []
    for tag in result.splitlines():
        version = tag_version(tag, prefix)
        if STABLE_VERSION_RE.match(version):
            tags.append(tag)
    return sorted(tags, key=lambda tag: semver_key(tag_version(tag, prefix)))


def latest_stable_tag(merged_ref: str, prefix: str, line: str | None = None) -> str:
    tags = stable_tags_merged(merged_ref, prefix, line)
    return tags[-1] if tags else ""


def remote_tag_target(tag: str) -> str:
    target = run_out(["git", "ls-remote", "--tags", "origin", f"refs/tags/{tag}^{{}}"], check=False)
    if not target:
        target = run_out(["git", "ls-remote", "--tags", "origin", f"refs/tags/{tag}"], check=False)
    return target.split()[0] if target else ""


def local_tag_target(tag: str) -> str:
    result = run(["git", "rev-parse", "--verify", "--quiet", f"{tag}^{{commit}}"], check=False)
    return result.stdout.strip() if result.returncode == 0 else ""


def create_or_push_tag(tag: str, target_sha: str, message: str) -> None:
    remote_target = remote_tag_target(tag)
    if remote_target:
        if remote_target != target_sha:
            raise RuntimeError(f"Tag {tag} already exists at {remote_target}, expected {target_sha}")
        log(f"Tag {tag} already exists at {target_sha}")
        return

    local_target = local_tag_target(tag)
    if local_target:
        if local_target == target_sha:
            log(f"Pushing existing local tag {tag}")
            run(["git", "push", "origin", tag])
            return
        log(f"Removing stale local tag {tag} at {local_target}")
        run(["git", "tag", "-d", tag])

    run(["git", "tag", "-a", tag, target_sha, "-m", message])
    run(["git", "push", "origin", tag])


def tag_message(tag: str) -> str:
    lines = run_out(["git", "for-each-ref", "--format=%(contents)", f"refs/tags/{tag}"]).splitlines()
    return lines[0] if lines else ""


def stable_tag_points_at(ref: str, prefix: str) -> str:
    result = run_out(["git", "tag", "--points-at", ref, "--list", f"{prefix}*"], check=False)
    tags = []
    for tag in result.splitlines():
        version = tag_version(tag, prefix)
        if STABLE_VERSION_RE.match(version):
            tags.append(tag)
    return sorted(tags, key=lambda tag: semver_key(tag_version(tag, prefix)))[-1] if tags else ""


def list_release_branches() -> list[str]:
    if RELEASE_BRANCH:
        return [RELEASE_BRANCH]

    result = run_out(
        ["git", "for-each-ref", "--format=%(refname:strip=3)", "refs/remotes/origin/release/*"],
        check=False,
    )
    return [line for line in result.splitlines() if line]


def branch_version_key(branch: str) -> tuple[int, int] | None:
    release_line = branch.removeprefix("release/").removeprefix("v")
    if not re.match(r"^[0-9]+\.[0-9]+$", release_line):
        return None
    major, minor = release_line.split(".")
    return int(major), int(minor)


def filter_supported_release_branches(branches: list[str]) -> list[str]:
    if RELEASE_BRANCH:
        return branches

    keyed = [(key, branch) for branch in branches if (key := branch_version_key(branch))]
    keyed.sort(key=lambda item: item[0])
    return [branch for _, branch in keyed[-SUPPORTED_RELEASE_BRANCH_COUNT:]]


def chart_yaml_value(ref: str, field: str) -> str:
    content = run_out(["git", "show", f"{ref}:charts/harbor-operator/Chart.yaml"])
    for line in content.splitlines():
        parts = line.split(None, 1)
        if len(parts) == 2 and parts[0] == f"{field}:":
            return parts[1].strip().strip('"')
    return ""


def set_chart_yaml_value(field: str, value: str) -> None:
    content = CHART_YAML.read_text()
    if field == "appVersion":
        replacement = f'appVersion: "{value}"'
    else:
        replacement = f"{field}: {value}"

    content = re.sub(rf"^{re.escape(field)}: .*$", replacement, content, flags=re.MULTILINE)
    CHART_YAML.write_text(content)


def wait_for_required_checks(sha: str) -> bool:
    if DRY_RUN:
        return True

    return wait_for_checks(
        sha,
        REQUIRED_CHECKS,
        timeout_seconds=1800,
        interval_seconds=30,
        logger=log,
    )


def github_release(tag: str) -> dict | None:
    result = run(
        ["gh", "release", "view", tag, "--json", "tagName,assets"],
        check=False,
    )
    if result.returncode != 0:
        return None
    return json.loads(result.stdout)


def operator_publish_complete(tag: str) -> bool:
    return github_release(tag) is not None


def chart_publish_complete(tag: str, version: str) -> bool:
    release = github_release(tag)
    if not release:
        return False

    expected_asset = f"harbor-operator-{version}.tgz"
    return any(asset.get("name") == expected_asset for asset in release.get("assets", []))


def schedule_operator_publish(version: str, tag: str) -> None:
    if operator_publish_complete(tag):
        log(f"Operator release {tag} already exists; no publish needed")
        return
    add_operator_release(version, tag, tag)


def schedule_chart_publish(version: str, operator_version: str, tag: str) -> None:
    if chart_publish_complete(tag, version):
        log(f"Chart release {tag} already exists with package asset; no publish needed")
        return
    add_chart_release(version, operator_version, tag, tag)


def changed_paths(base: str, head: str, pathspecs: tuple[str, ...] = ()) -> list[str]:
    args = ["git", "diff", "--name-only", f"{base}..{head}"]
    if pathspecs:
        args.extend(["--", *pathspecs])
    return run_out(args).splitlines()


def add_operator_release(version: str, tag: str, ref: str) -> None:
    operator_releases.append({"version": version, "tag": tag, "ref": ref})


def add_chart_release(version: str, operator_version: str, tag: str, ref: str) -> None:
    chart_releases.append(
        {"version": version, "operator_version": operator_version, "tag": tag, "ref": ref}
    )


def write_outputs() -> None:
    operator_matrix = json.dumps({"include": operator_releases}, separators=(",", ":"))
    chart_matrix = json.dumps({"include": chart_releases}, separators=(",", ":"))

    log(f"Operator publish matrix: {operator_matrix}")
    log(f"Chart publish matrix: {chart_matrix}")

    if not GITHUB_OUTPUT:
        return

    with open(GITHUB_OUTPUT, "a", encoding="utf-8") as output:
        output.write(f"operator_matrix={operator_matrix}\n")
        output.write(f"chart_matrix={chart_matrix}\n")
        output.write(f"has_operator_releases={str(bool(operator_releases)).lower()}\n")
        output.write(f"has_chart_releases={str(bool(chart_releases)).lower()}\n")


def process_branch(branch: str) -> None:
    log(f"Inspecting {branch}")

    if not BRANCH_RE.match(branch):
        log(f"Skipping {branch}: expected branch name format release/v<major>.<minor>")
        return

    remote_ref = f"origin/{branch}"
    if run(["git", "rev-parse", "--verify", "--quiet", remote_ref], check=False).returncode != 0:
        log(f"Skipping {branch}: {remote_ref} does not exist")
        return

    release_line = branch.removeprefix("release/").removeprefix("v")
    head_sha = run_out(["git", "rev-parse", remote_ref])
    operator_tag = latest_stable_tag(remote_ref, "v", release_line)
    if not operator_tag:
        log(f"Skipping {branch}: no stable operator tag found for v{release_line}.x")
        return

    last_release_sha = run_out(["git", "rev-list", "-n", "1", operator_tag])
    chart_tag_on_head = stable_tag_points_at(head_sha, "chart-v")

    if head_sha == last_release_sha:
        complete_missing_chart_release(branch, remote_ref, head_sha, operator_tag, chart_tag_on_head)
        return

    operator_changed_paths = changed_paths(operator_tag, remote_ref, OPERATOR_RUNTIME_PATHS)
    chart_tag = latest_stable_tag(remote_ref, "chart-v")
    chart_base = chart_tag or operator_tag
    chart_changed_paths = changed_paths(chart_base, remote_ref, (CHART_PATH,))

    if not operator_changed_paths and not chart_changed_paths:
        log(f"Skipping {branch}: no operator or chart changes requiring automated release")
        return

    manual_operator_paths = [
        path for path in operator_changed_paths if path not in AUTO_RELEASE_TRIGGER_PATHS
    ]
    if manual_operator_paths:
        log(
            f"Skipping {branch}: operator code changes require explicit release intent: "
            f"{', '.join(manual_operator_paths[:5])}"
        )
        return

    unreleased_chart_paths = [
        path for path in chart_changed_paths if path != CHART_YAML_PATH
    ]
    if unreleased_chart_paths:
        log(
            f"Skipping {branch}: chart changes require explicit chart release intent: "
            f"{', '.join(unreleased_chart_paths[:5])}"
        )
        return

    auto_release_change_detected = any(
        path in AUTO_RELEASE_TRIGGER_PATHS for path in operator_changed_paths
    )
    if not auto_release_change_detected:
        log(f"Skipping {branch}: no auto-release-relevant dependency changes since {operator_tag}")
        return

    create_dependency_patch_release(
        branch=branch,
        remote_ref=remote_ref,
        head_sha=head_sha,
        operator_tag=operator_tag,
        chart_yaml_changed=CHART_YAML_PATH in chart_changed_paths,
    )


def complete_missing_chart_release(
    branch: str,
    remote_ref: str,
    head_sha: str,
    operator_tag: str,
    chart_tag_on_head: str,
) -> None:
    if not tag_message(operator_tag).startswith("Automated dependency patch release"):
        log(
            f"Skipping {branch}: no commits since {operator_tag} "
            "and the operator tag is not from an automated patch release"
        )
        return

    chart_version = chart_yaml_value(remote_ref, "version")
    chart_operator_version = chart_yaml_value(remote_ref, "appVersion")
    operator_version = tag_version(operator_tag, "v")
    if chart_operator_version != operator_version:
        log(
            f"Chart.yaml appVersion {chart_operator_version} does not match {operator_tag}; "
            "skipping chart backfill"
        )
        if operator_publish_complete(operator_tag):
            log(f"Automated operator patch {operator_tag} is already published")
            return
        if not wait_for_required_checks(head_sha):
            log(f"Skipping {branch}: required checks are not green on {head_sha}")
            return
        add_operator_release(operator_version, operator_tag, operator_tag)
        return

    if not STABLE_VERSION_RE.match(chart_version):
        log(f"Skipping {branch}: Chart.yaml version {chart_version} is not a stable chart version")
        return

    chart_tag = f"chart-v{chart_version}"
    existing_chart_tag_target = remote_tag_target(chart_tag)
    if existing_chart_tag_target and existing_chart_tag_target != head_sha:
        log(f"Skipping {branch}: tag {chart_tag} already exists at {existing_chart_tag_target}")
        return

    if chart_tag_on_head and chart_tag_on_head != chart_tag:
        log(f"Skipping {branch}: chart tag {chart_tag_on_head} points at HEAD, expected {chart_tag}")
        return

    operator_needs_publish = not operator_publish_complete(operator_tag)
    chart_needs_publish = not (
        existing_chart_tag_target and chart_publish_complete(chart_tag, chart_version)
    )

    if not operator_needs_publish and not chart_needs_publish:
        log(f"Automated operator patch {operator_tag} and chart patch {chart_tag} are already published")
        return

    if not operator_needs_publish:
        log(f"Automated operator patch {operator_tag} is already published")
    if not chart_needs_publish:
        log(f"Automated chart patch {chart_tag} is already published")

    log(f"Completing automated patch publish for {operator_tag} / {chart_tag}")

    if not wait_for_required_checks(head_sha):
        log(f"Skipping {branch}: required checks are not green on {head_sha}")
        return

    if operator_needs_publish:
        add_operator_release(operator_version, operator_tag, operator_tag)

    if chart_needs_publish and not DRY_RUN and not existing_chart_tag_target:
        create_or_push_tag(chart_tag, head_sha, f"Automated chart patch release {chart_tag} for {operator_tag}")

    if chart_needs_publish:
        add_chart_release(chart_version, operator_version, chart_tag, chart_tag)


def create_dependency_patch_release(
    *,
    branch: str,
    remote_ref: str,
    head_sha: str,
    operator_tag: str,
    chart_yaml_changed: bool,
) -> None:
    current_operator_version = tag_version(operator_tag, "v")
    next_operator_version = increment_patch(current_operator_version)
    next_operator_tag = f"v{next_operator_version}"

    chart_tag = latest_stable_tag(remote_ref, "chart-v")
    if chart_tag:
        current_chart_version = tag_version(chart_tag, "chart-v")
    else:
        current_chart_version = chart_yaml_value(operator_tag, "version")

    next_chart_version = increment_patch(current_chart_version)
    next_chart_tag = f"chart-v{next_chart_version}"

    current_head_chart_version = chart_yaml_value(remote_ref, "version")
    current_head_operator_version = chart_yaml_value(remote_ref, "appVersion")
    target_sha = head_sha

    log(f"Branch {branch} is eligible")
    log(f"Next operator tag: {next_operator_tag}")
    log(f"Next chart tag: {next_chart_tag}")

    if (
        current_head_operator_version == next_operator_version
        and current_head_chart_version == next_chart_version
    ):
        log("Chart.yaml already contains automated patch metadata")
    elif (
        current_head_operator_version == current_operator_version
        and current_head_chart_version == current_chart_version
    ) or not chart_yaml_changed:
        if (
            current_head_operator_version != current_operator_version
            or current_head_chart_version != current_chart_version
        ):
            log(
                "Normalizing stale Chart.yaml metadata from "
                f"{current_head_chart_version}/{current_head_operator_version}"
            )

        if DRY_RUN:
            log(
                "Would update Chart.yaml to version "
                f"{next_chart_version} and appVersion {next_operator_version}"
            )
        else:
            run(["git", "switch", "-C", branch, remote_ref])
            set_chart_yaml_value("version", next_chart_version)
            set_chart_yaml_value("appVersion", next_operator_version)
            run(["git", "add", str(CHART_YAML)])
            run(["git", "commit", "-m", f"chore(release): prepare {next_operator_tag} and {next_chart_tag}"])
            run(["git", "push", "origin", f"HEAD:refs/heads/{branch}"])
            target_sha = run_out(["git", "rev-parse", "HEAD"])
    else:
        log(
            f"Skipping {branch}: Chart.yaml has unexpected release metadata "
            f"{current_head_chart_version}/{current_head_operator_version}"
        )
        log(
            f"Expected either {current_chart_version}/{current_operator_version} "
            f"or {next_chart_version}/{next_operator_version}"
        )
        return

    if DRY_RUN:
        add_operator_release(next_operator_version, next_operator_tag, next_operator_tag)
        add_chart_release(next_chart_version, next_operator_version, next_chart_tag, next_chart_tag)
        return

    if not wait_for_required_checks(target_sha):
        log(f"Skipping {branch}: required checks are not green on {target_sha}")
        return

    existing_operator_tag_target = remote_tag_target(next_operator_tag)
    if existing_operator_tag_target:
        if existing_operator_tag_target != target_sha:
            log(
                f"Skipping {branch}: operator tag {next_operator_tag} "
                f"already exists at {existing_operator_tag_target}"
            )
            return
        log(f"Operator tag {next_operator_tag} already exists at HEAD")
    else:
        create_or_push_tag(
            next_operator_tag,
            target_sha,
            f"Automated dependency patch release {next_operator_tag}",
        )

    existing_chart_tag_target = remote_tag_target(next_chart_tag)
    if existing_chart_tag_target:
        if existing_chart_tag_target != target_sha:
            log(
                f"Skipping {branch}: chart tag {next_chart_tag} "
                f"already exists at {existing_chart_tag_target}"
            )
            return
        log(f"Chart tag {next_chart_tag} already exists at HEAD")
    else:
        create_or_push_tag(
            next_chart_tag,
            target_sha,
            f"Automated chart patch release {next_chart_tag} for {next_operator_tag}",
        )

    schedule_operator_publish(next_operator_version, next_operator_tag)
    schedule_chart_publish(next_chart_version, next_operator_version, next_chart_tag)


def main() -> int:
    run(["git", "fetch", "origin", "+refs/heads/*:refs/remotes/origin/*", "--tags", "--prune"])

    branches = list_release_branches()
    if not branches:
        log("No release branches found")
        return 0

    if not RELEASE_BRANCH:
        branches = filter_supported_release_branches(branches)
        if not branches:
            log("No supported release branches found")
            return 0
        log(f"Processing supported release branches: {' '.join(branches)}")

    run(["git", "config", "user.name", "github-actions[bot]"])
    run(["git", "config", "user.email", "41898282+github-actions[bot]@users.noreply.github.com"])

    for branch in branches:
        process_branch(branch)

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    finally:
        write_outputs()
