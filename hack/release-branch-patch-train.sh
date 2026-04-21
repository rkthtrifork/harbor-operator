#!/usr/bin/env bash
set -euo pipefail

DRY_RUN="${DRY_RUN:-false}"
RELEASE_BRANCH="${RELEASE_BRANCH:-}"
GHCR_IMAGE="${GHCR_IMAGE:-}"
GHCR_USERNAME="${GHCR_USERNAME:-}"
GHCR_TOKEN="${GHCR_TOKEN:-}"
IMAGE_WAIT_TIMEOUT_SECONDS="${IMAGE_WAIT_TIMEOUT_SECONDS:-900}"
IMAGE_WAIT_INTERVAL_SECONDS="${IMAGE_WAIT_INTERVAL_SECONDS:-15}"
SUPPORTED_RELEASE_BRANCH_COUNT="${SUPPORTED_RELEASE_BRANCH_COUNT:-3}"

AUTO_RELEASE_PATHS=(
	"go.mod"
	"go.sum"
	"Dockerfile"
)

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

tag_message() {
	local tag="$1"
	git for-each-ref --format='%(contents)' "refs/tags/${tag}" | head -n1
}

stable_tag_points_at() {
	local ref="$1"
	local prefix="$2"

	(git tag --points-at "$ref" --list "${prefix}*" |
		grep -E "^${prefix}[0-9]+\.[0-9]+\.[0-9]+$" |
		sort -V |
		tail -n1) || true
}

ensure_ghcr_login() {
	if [[ -z "$GHCR_IMAGE" || -z "$GHCR_USERNAME" || -z "$GHCR_TOKEN" ]]; then
		log "Skipping image availability checks: GHCR credentials or image name not configured"
		return 1
	fi

	echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin >/dev/null
}

wait_for_image_tag() {
	local image_tag="$1"
	local attempt=0
	local max_attempts=$((IMAGE_WAIT_TIMEOUT_SECONDS / IMAGE_WAIT_INTERVAL_SECONDS))

	if (( max_attempts < 1 )); then
		max_attempts=1
	fi

	while (( attempt < max_attempts )); do
		if docker manifest inspect "${GHCR_IMAGE}:${image_tag}" >/dev/null 2>&1; then
			log "Found published operator image ${GHCR_IMAGE}:${image_tag}"
			return 0
		fi

		attempt=$((attempt + 1))
		log "Waiting for operator image ${GHCR_IMAGE}:${image_tag} (${attempt}/${max_attempts})"
		sleep "$IMAGE_WAIT_INTERVAL_SECONDS"
	done

	log "Timed out waiting for operator image ${GHCR_IMAGE}:${image_tag}"
	return 1
}

list_release_branches() {
	if [[ -n "$RELEASE_BRANCH" ]]; then
		printf '%s\n' "$RELEASE_BRANCH"
		return
	fi

	git for-each-ref --format='%(refname:strip=3)' refs/remotes/origin/release/*
}

branch_version_key() {
	local branch="$1"
	local release_line="${branch#release/}"
	release_line="${release_line#v}"

	if [[ ! "$release_line" =~ ^[0-9]+\.[0-9]+$ ]]; then
		return 1
	fi

	printf '%s\n' "$release_line"
}

filter_supported_release_branches() {
	local branches=("$@")
	local branch
	local line

	if [[ -n "$RELEASE_BRANCH" ]]; then
		printf '%s\n' "${branches[@]}"
		return
	fi

	for branch in "${branches[@]}"; do
		if line="$(branch_version_key "$branch")"; then
			printf '%s %s\n' "$line" "$branch"
		fi
	done |
		sort -k1,1V |
		tail -n "$SUPPORTED_RELEASE_BRANCH_COUNT" |
		awk '{print $2}'
}

path_is_auto_release_relevant() {
	local path="$1"
	local allowed_path

	for allowed_path in "${AUTO_RELEASE_PATHS[@]}"; do
		if [[ "$path" == "$allowed_path" ]]; then
			return 0
		fi
	done

	return 1
}

git fetch origin '+refs/heads/*:refs/remotes/origin/*' --tags --prune

mapfile -t branches < <(list_release_branches)

if [[ "${#branches[@]}" -eq 0 ]]; then
	log "No release branches found"
	exit 0
fi

if [[ -z "$RELEASE_BRANCH" ]]; then
	mapfile -t branches < <(filter_supported_release_branches "${branches[@]}")

	if [[ "${#branches[@]}" -eq 0 ]]; then
		log "No supported release branches found"
		exit 0
	fi

	log "Processing supported release branches: ${branches[*]}"
fi

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"

if [[ "$DRY_RUN" != "true" ]]; then
	ensure_ghcr_login
fi

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
	operator_tag_on_head="$(stable_tag_points_at "${head_sha}" "v")"
	chart_tag_on_head="$(stable_tag_points_at "${head_sha}" "chart-v")"

	chart_tag="$( (git tag --merged "${remote_ref}" --list 'chart-v*' | grep -E '^chart-v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n1) || true )"
	if [[ -n "$chart_tag" ]]; then
		current_chart_version="${chart_tag#chart-v}"
	else
		current_chart_version="$(git show "${remote_ref}:charts/harbor-operator/Chart.yaml" | awk '/^version:/ {print $2; exit}')"
	fi

	if [[ "$head_sha" == "$last_release_sha" ]]; then
		if [[ -n "$chart_tag_on_head" ]]; then
			log "Skipping ${branch}: no commits since ${operator_tag} and chart tag ${chart_tag_on_head} already points at HEAD"
			continue
		fi

		if [[ "$(tag_message "${operator_tag}")" != Automated\ dependency\ patch\ release* ]]; then
			log "Skipping ${branch}: no commits since ${operator_tag} and missing chart tag is not from an automated patch release"
			continue
		fi

		current_operator_version="${operator_tag#v}"
		next_chart_version="$(increment_patch "$current_chart_version")"
		next_chart_tag="chart-v${next_chart_version}"

		if tag_exists_remote "$next_chart_tag"; then
			log "Skipping ${branch}: tag ${next_chart_tag} already exists"
			continue
		fi

		log "Branch ${branch} is completing an in-progress automated patch release"
		log "Existing operator tag: ${operator_tag}"
		log "Next chart tag: ${next_chart_tag}"

		if [[ "$DRY_RUN" == "true" ]]; then
			continue
		fi

		wait_for_image_tag "${current_operator_version}"
		git tag -a "${next_chart_tag}" "${head_sha}" -m "Automated chart patch release ${next_chart_tag} for ${operator_tag}"
		git push origin "${next_chart_tag}"
		continue
	fi

	mapfile -t changed_paths < <(git diff --name-only "${operator_tag}..${remote_ref}")
	if [[ "${#changed_paths[@]}" -eq 0 ]]; then
		log "Skipping ${branch}: no file changes since ${operator_tag}"
		continue
	fi

	auto_release_change_detected=false
	for path in "${changed_paths[@]}"; do
		if path_is_auto_release_relevant "$path"; then
			auto_release_change_detected=true
			break
		fi
	done

	if [[ "$auto_release_change_detected" != true ]]; then
		log "Skipping ${branch}: no auto-release-relevant dependency changes since ${operator_tag}"
		continue
	fi

	current_operator_version="${operator_tag#v}"
	next_operator_version="$(increment_patch "$current_operator_version")"
	next_operator_tag="v${next_operator_version}"

	next_chart_version="$(increment_patch "$current_chart_version")"
	next_chart_tag="chart-v${next_chart_version}"

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

	if tag_exists_remote "$next_operator_tag"; then
		log "Operator tag ${next_operator_tag} already exists; resuming with image availability checks"
	else
		git tag -a "${next_operator_tag}" "${head_sha}" -m "Automated dependency patch release ${next_operator_tag}"
		git push origin "${next_operator_tag}"
	fi

	wait_for_image_tag "${next_operator_version}"
	git tag -a "${next_chart_tag}" "${head_sha}" -m "Automated chart patch release ${next_chart_tag} for ${next_operator_tag}"
	git push origin "${next_chart_tag}"
done
