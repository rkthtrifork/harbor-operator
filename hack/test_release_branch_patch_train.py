#!/usr/bin/env python3
"""Local tests for release_branch_patch_train.py.

These tests use temporary git repositories and dry-run mode, so they do not
touch the real remote or GitHub API.
"""

from __future__ import annotations

import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
SCRIPT = REPO_ROOT / "hack" / "release_branch_patch_train.py"


def run(args: list[str], cwd: Path, *, env: dict[str, str] | None = None) -> str:
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    return subprocess.run(
        args,
        cwd=cwd,
        env=merged_env,
        check=True,
        text=True,
        capture_output=True,
    ).stdout


class PatchTrainRepo:
    def __init__(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.origin = self.root / "origin.git"
        self.work = self.root / "work"
        run(["git", "init", "--bare", str(self.origin)], self.root)
        run(["git", "clone", str(self.origin), str(self.work)], self.root)
        run(["git", "config", "user.name", "test"], self.work)
        run(["git", "config", "user.email", "test@example.com"], self.work)
        self._seed()

    def cleanup(self) -> None:
        self.tmp.cleanup()

    def _seed(self) -> None:
        for directory in (
            "api",
            "cmd",
            "internal",
            "charts/harbor-operator/templates",
        ):
            (self.work / directory).mkdir(parents=True, exist_ok=True)

        (self.work / "charts/harbor-operator/Chart.yaml").write_text(
            'apiVersion: v2\nname: harbor-operator\nversion: 0.5.0\nappVersion: "0.7.0"\n'
        )
        (self.work / "charts/harbor-operator/templates/deployment.yaml").write_text("old chart\n")
        (self.work / "go.mod").write_text("module example.com/test\n")
        (self.work / "go.sum").write_text("")
        (self.work / "Dockerfile").write_text("FROM scratch\n")
        (self.work / "api/a.go").write_text("package api\n")
        (self.work / "cmd/main.go").write_text("package main\nfunc main(){}\n")
        (self.work / "internal/i.go").write_text("package internal\n")

        self.commit("initial release metadata")
        run(["git", "branch", "-M", "main"], self.work)
        self.tag("v0.7.0", "Release v0.7.0")
        self.tag("chart-v0.5.0", "Chart release chart-v0.5.0 for v0.7.0")
        run(["git", "push", "origin", "main", "--tags"], self.work)
        run(["git", "switch", "-c", "release/v0.7"], self.work)
        run(["git", "push", "origin", "release/v0.7"], self.work)

    def commit(self, message: str) -> None:
        run(["git", "add", "."], self.work)
        run(["git", "commit", "-m", message], self.work)

    def tag(self, tag: str, message: str) -> None:
        run(["git", "tag", "-a", tag, "-m", message], self.work)

    def push_branch_and_tags(self) -> None:
        run(["git", "push", "origin", "release/v0.7", "--tags"], self.work)

    def patch_train_outputs(self) -> dict[str, str]:
        output = self.root / "outputs"
        run(
            ["python3", str(SCRIPT)],
            self.work,
            env={
                "DRY_RUN": "true",
                "RELEASE_BRANCH": "release/v0.7",
                "GITHUB_OUTPUT": str(output),
            },
        )
        return dict(line.split("=", 1) for line in output.read_text().splitlines())


class ReleaseBranchPatchTrainTests(unittest.TestCase):
    def setUp(self) -> None:
        self.repo = PatchTrainRepo()

    def tearDown(self) -> None:
        self.repo.cleanup()

    def test_dependency_update_schedules_operator_and_chart(self) -> None:
        with (self.repo.work / "go.sum").open("a") as go_sum:
            go_sum.write("# dep update\n")
        self.repo.commit("chore(deps): update dependency")
        self.repo.push_branch_and_tags()

        outputs = self.repo.patch_train_outputs()

        self.assertEqual(outputs["has_operator_releases"], "true")
        self.assertEqual(outputs["has_chart_releases"], "true")
        self.assertEqual(
            json.loads(outputs["operator_matrix"])["include"][0]["tag"],
            "v0.7.1",
        )
        self.assertEqual(
            json.loads(outputs["chart_matrix"])["include"][0]["tag"],
            "chart-v0.5.1",
        )

    def test_prior_chart_only_release_does_not_block_dependency_update(self) -> None:
        (self.repo.work / "charts/harbor-operator/Chart.yaml").write_text(
            'apiVersion: v2\nname: harbor-operator\nversion: 0.5.1\nappVersion: "0.7.0"\n'
        )
        (self.repo.work / "charts/harbor-operator/templates/deployment.yaml").write_text(
            "chart-only change\n"
        )
        self.repo.commit("chore(chart): release chart patch")
        self.repo.tag("chart-v0.5.1", "Chart release chart-v0.5.1 for v0.7.0")
        self.repo.push_branch_and_tags()

        with (self.repo.work / "go.sum").open("a") as go_sum:
            go_sum.write("# dep update\n")
        self.repo.commit("chore(deps): update dependency")
        self.repo.push_branch_and_tags()

        outputs = self.repo.patch_train_outputs()

        self.assertEqual(outputs["has_operator_releases"], "true")
        self.assertEqual(
            json.loads(outputs["chart_matrix"])["include"][0]["tag"],
            "chart-v0.5.2",
        )

    def test_unpublished_chart_change_blocks_dependency_update(self) -> None:
        (self.repo.work / "charts/harbor-operator/templates/deployment.yaml").write_text(
            "unpublished chart change\n"
        )
        with (self.repo.work / "go.sum").open("a") as go_sum:
            go_sum.write("# dep update\n")
        self.repo.commit("mixed chart and dependency change")
        self.repo.push_branch_and_tags()

        outputs = self.repo.patch_train_outputs()

        self.assertEqual(outputs["has_operator_releases"], "false")
        self.assertEqual(outputs["has_chart_releases"], "false")

    def test_existing_automated_tags_schedule_missing_publishes(self) -> None:
        (self.repo.work / "charts/harbor-operator/Chart.yaml").write_text(
            'apiVersion: v2\nname: harbor-operator\nversion: 0.5.1\nappVersion: "0.7.1"\n'
        )
        with (self.repo.work / "go.sum").open("a") as go_sum:
            go_sum.write("# dep update\n")
        self.repo.commit("chore(release): prepare v0.7.1 and chart-v0.5.1")
        self.repo.tag("v0.7.1", "Automated dependency patch release v0.7.1")
        self.repo.tag("chart-v0.5.1", "Automated chart patch release chart-v0.5.1 for v0.7.1")
        self.repo.push_branch_and_tags()

        outputs = self.repo.patch_train_outputs()

        self.assertEqual(outputs["has_operator_releases"], "true")
        self.assertEqual(outputs["has_chart_releases"], "true")
        self.assertEqual(
            json.loads(outputs["operator_matrix"])["include"][0]["tag"],
            "v0.7.1",
        )
        self.assertEqual(
            json.loads(outputs["chart_matrix"])["include"][0]["tag"],
            "chart-v0.5.1",
        )

    def test_stale_chart_metadata_does_not_block_operator_recovery(self) -> None:
        with (self.repo.work / "go.sum").open("a") as go_sum:
            go_sum.write("# dep update\n")
        self.repo.commit("automated dependency patch without chart metadata")
        self.repo.tag("v0.7.1", "Automated dependency patch release v0.7.1")
        self.repo.push_branch_and_tags()

        outputs = self.repo.patch_train_outputs()

        self.assertEqual(outputs["has_operator_releases"], "true")
        self.assertEqual(outputs["has_chart_releases"], "false")
        self.assertEqual(
            json.loads(outputs["operator_matrix"])["include"][0]["tag"],
            "v0.7.1",
        )


if __name__ == "__main__":
    unittest.main()
