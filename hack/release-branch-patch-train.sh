#!/usr/bin/env bash
set -euo pipefail

if ! command -v gh >/dev/null 2>&1; then
	echo "gh CLI is required" >&2
	exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
	echo "jq is required" >&2
	exit 1
fi

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must be set}"
: "${GITHUB_TOKEN:?GITHUB_TOKEN must be set}"

AUTO_RELEASE_LABEL="${AUTO_RELEASE_LABEL:-dependencies}"
DRY_RUN="${DRY_RUN:-false}"
RELEASE_BRANCH="${RELEASE_BRANCH:-}"

log() {
	echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] $*"
}

increment_patch() {
	local version="$1"
	IFS=. read -r major minor patch <<<"$version"
	printf '%s.%s.%s\n' "$major" "$minor" "$((patch + 1))"
}

latest_stable_tag() {
	local merged_ref="$1"
	local prefix="$2"
	local line="$3"
	local escaped_line="${line//./\\.}"

	(git tag --merged "$merged_ref" --list "${prefix}${line}.*" |
		grep -E "^${prefix}${escaped_line}\\.[0-9]+$" |
		sort -V |
		tail -n1) || true
}

tag_exists_remote() {
	local tag="$1"
	git ls-remote --tags origin "refs/tags/${tag}" | grep -q .
}

list_release_branches() {
	if [[ -n "$RELEASE_BRANCH" ]]; then
		printf '%s\n' "$RELEASE_BRANCH"
		return
	fi

	git for-each-ref --format='%(refname:strip=3)' refs/remotes/origin/release/*
}

commit_pr_number() {
	local sha="$1"
	local branch="$2"
	local prs_json

	prs_json="$(gh api "repos/${GITHUB_REPOSITORY}/commits/${sha}/pulls" -H "Accept: application/vnd.github+json")"
	jq -r --arg branch "$branch" 'map(select(.base.ref == $branch))[0].number // empty' <<<"$prs_json"
}

pr_has_release_label() {
	local pr_json="$1"
	jq -e --arg label "$AUTO_RELEASE_LABEL" '.labels | map(.name) | index($label) != null' <<<"$pr_json" >/dev/null
}

git fetch origin '+refs/heads/*:refs/remotes/origin/*' --tags --prune

mapfile -t branches < <(list_release_branches)

if [[ "${#branches[@]}" -eq 0 ]]; then
	log "No release branches found"
	exit 0
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

for branch in "${branches[@]}"; do
	log "Inspecting ${branch}"

	if [[ ! "$branch" =~ ^release/(v)?[0-9]+\.[0-9]+$ ]]; then
		log "Skipping ${branch}: expected branch name format release/v<major>.<minor>"
		continue
	fi

	release_line="${branch#release/}"
	release_line="${release_line#v}"
	remote_ref="origin/${branch}"
	head_sha="$(git rev-parse "${remote_ref}")"

	operator_tag="$(latest_stable_tag "${remote_ref}" "v" "${release_line}")"
	if [[ -z "$operator_tag" ]]; then
		log "Skipping ${branch}: no stable operator tag found for v${release_line}.x"
		continue
	fi

	last_release_sha="$(git rev-list -n 1 "${operator_tag}")"
	if [[ "$head_sha" == "$last_release_sha" ]]; then
		log "Skipping ${branch}: no commits since ${operator_tag}"
		continue
	fi

	mapfile -t commits < <(git rev-list --reverse "${operator_tag}..${remote_ref}")
	if [[ "${#commits[@]}" -eq 0 ]]; then
		log "Skipping ${branch}: no commits since ${operator_tag}"
		continue
	fi

	declare -A seen_prs=()
	eligible=true

	for sha in "${commits[@]}"; do
		pr_number="$(commit_pr_number "$sha" "$branch")"
		if [[ -z "$pr_number" ]]; then
			log "Skipping ${branch}: commit ${sha} is not associated with a pull request targeting ${branch}"
			eligible=false
			break
		fi

		if [[ -n "${seen_prs[$pr_number]:-}" ]]; then
			continue
		fi

		seen_prs["$pr_number"]=1
		pr_json="$(gh api "repos/${GITHUB_REPOSITORY}/pulls/${pr_number}")"
		pr_title="$(jq -r '.title' <<<"$pr_json")"

		if ! pr_has_release_label "$pr_json"; then
			log "Skipping ${branch}: PR #${pr_number} (${pr_title}) does not have the ${AUTO_RELEASE_LABEL} label"
			eligible=false
			break
		fi
	done

	if [[ "$eligible" != true ]]; then
		continue
	fi

	current_operator_version="${operator_tag#v}"
	next_operator_version="$(increment_patch "$current_operator_version")"
	next_operator_tag="v${next_operator_version}"

	chart_tag="$( (git tag --merged "${remote_ref}" --list 'chart-v*' | grep -E '^chart-v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n1) || true )"
	if [[ -n "$chart_tag" ]]; then
		current_chart_version="${chart_tag#chart-v}"
	else
		current_chart_version="$(git show "${remote_ref}:charts/harbor-operator/Chart.yaml" | awk '/^version:/ {print $2; exit}')"
	fi
	next_chart_version="$(increment_patch "$current_chart_version")"
	next_chart_tag="chart-v${next_chart_version}"

	if tag_exists_remote "$next_operator_tag"; then
		log "Skipping ${branch}: tag ${next_operator_tag} already exists"
		continue
	fi

	if tag_exists_remote "$next_chart_tag"; then
		log "Skipping ${branch}: tag ${next_chart_tag} already exists"
		continue
	fi

	log "Branch ${branch} is eligible"
	log "Next operator tag: ${next_operator_tag}"
	log "Next chart tag: ${next_chart_tag}"

	if [[ "$DRY_RUN" == "true" ]]; then
		continue
	fi

	git tag -a "${next_operator_tag}" "${head_sha}" -m "Automated dependency patch release ${next_operator_tag}"
	git tag -a "${next_chart_tag}" "${head_sha}" -m "Automated chart patch release ${next_chart_tag} for ${next_operator_tag}"
	git push origin "${next_operator_tag}" "${next_chart_tag}"
done
