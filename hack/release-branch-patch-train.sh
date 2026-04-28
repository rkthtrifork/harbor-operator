#!/usr/bin/env bash
set -euo pipefail

DRY_RUN="${DRY_RUN:-false}"
RELEASE_BRANCH="${RELEASE_BRANCH:-}"
SUPPORTED_RELEASE_BRANCH_COUNT="${SUPPORTED_RELEASE_BRANCH_COUNT:-3}"
GITHUB_OUTPUT="${GITHUB_OUTPUT:-}"

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

remote_tag_target() {
	local tag="$1"
	local target

	target="$(git ls-remote --tags origin "refs/tags/${tag}^{}" | awk '{print $1}')"
	if [[ -z "$target" ]]; then
		target="$(git ls-remote --tags origin "refs/tags/${tag}" | awk '{print $1}')"
	fi

	printf '%s\n' "$target"
}

local_tag_target() {
	local tag="$1"

	if git rev-parse --verify --quiet "${tag}^{commit}" >/dev/null; then
		git rev-parse "${tag}^{commit}"
	fi
}

create_or_push_tag() {
	local tag="$1"
	local target_sha="$2"
	local message="$3"
	local remote_target
	local local_target

	remote_target="$(remote_tag_target "$tag")"
	if [[ -n "$remote_target" ]]; then
		if [[ "$remote_target" != "$target_sha" ]]; then
			log "Tag ${tag} already exists at ${remote_target}, expected ${target_sha}"
			return 1
		fi

		log "Tag ${tag} already exists at ${target_sha}"
		return 0
	fi

	local_target="$(local_tag_target "$tag")"
	if [[ -n "$local_target" ]]; then
		if [[ "$local_target" == "$target_sha" ]]; then
			log "Pushing existing local tag ${tag}"
			git push origin "$tag"
			return 0
		fi

		log "Removing stale local tag ${tag} at ${local_target}"
		git tag -d "$tag" >/dev/null
	fi

	git tag -a "$tag" "$target_sha" -m "$message"
	git push origin "$tag"
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

operator_releases_file="$(mktemp)"
chart_releases_file="$(mktemp)"
printf '[]\n' > "$operator_releases_file"
printf '[]\n' > "$chart_releases_file"

add_operator_release() {
	local version="$1"
	local tag="$2"
	local ref="$3"

	jq \
		--arg version "$version" \
		--arg tag "$tag" \
		--arg ref "$ref" \
		'. += [{version: $version, tag: $tag, ref: $ref}]' \
		"$operator_releases_file" > "${operator_releases_file}.tmp"
	mv "${operator_releases_file}.tmp" "$operator_releases_file"
}

add_chart_release() {
	local version="$1"
	local operator_version="$2"
	local tag="$3"
	local ref="$4"

	jq \
		--arg version "$version" \
		--arg operator_version "$operator_version" \
		--arg tag "$tag" \
		--arg ref "$ref" \
		'. += [{version: $version, operator_version: $operator_version, tag: $tag, ref: $ref}]' \
		"$chart_releases_file" > "${chart_releases_file}.tmp"
	mv "${chart_releases_file}.tmp" "$chart_releases_file"
}

write_outputs() {
	local operator_matrix
	local chart_matrix

	operator_matrix="$(jq -c '{include: .}' "$operator_releases_file")"
	chart_matrix="$(jq -c '{include: .}' "$chart_releases_file")"

	log "Operator publish matrix: ${operator_matrix}"
	log "Chart publish matrix: ${chart_matrix}"

	if [[ -n "$GITHUB_OUTPUT" ]]; then
		{
			echo "operator_matrix=${operator_matrix}"
			echo "chart_matrix=${chart_matrix}"
			if jq -e 'length > 0' "$operator_releases_file" >/dev/null; then
				echo "has_operator_releases=true"
			else
				echo "has_operator_releases=false"
			fi
			if jq -e 'length > 0' "$chart_releases_file" >/dev/null; then
				echo "has_chart_releases=true"
			else
				echo "has_chart_releases=false"
			fi
		} >> "$GITHUB_OUTPUT"
	fi
}

trap write_outputs EXIT

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
			add_chart_release "$next_chart_version" "$current_operator_version" "$next_chart_tag" "$next_chart_tag"
			continue
		fi

		create_or_push_tag "${next_chart_tag}" "${head_sha}" "Automated chart patch release ${next_chart_tag} for ${operator_tag}"
		add_chart_release "$next_chart_version" "$current_operator_version" "$next_chart_tag" "$next_chart_tag"
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

	existing_chart_tag_target="$(remote_tag_target "$next_chart_tag")"
	if [[ -n "$existing_chart_tag_target" ]]; then
		if [[ "$existing_chart_tag_target" == "$head_sha" ]]; then
			log "Chart tag ${next_chart_tag} already exists at HEAD; scheduling chart publish"
			add_chart_release "$next_chart_version" "$next_operator_version" "$next_chart_tag" "$next_chart_tag"
		else
			log "Skipping ${branch}: tag ${next_chart_tag} already exists at ${existing_chart_tag_target}"
		fi
		continue
	fi

	log "Branch ${branch} is eligible"
	log "Next operator tag: ${next_operator_tag}"
	log "Next chart tag: ${next_chart_tag}"

	if [[ "$DRY_RUN" == "true" ]]; then
		add_operator_release "$next_operator_version" "$next_operator_tag" "$next_operator_tag"
		add_chart_release "$next_chart_version" "$next_operator_version" "$next_chart_tag" "$next_chart_tag"
		continue
	fi

	existing_operator_tag_target="$(remote_tag_target "$next_operator_tag")"
	if [[ -n "$existing_operator_tag_target" ]]; then
		if [[ "$existing_operator_tag_target" != "$head_sha" ]]; then
			log "Skipping ${branch}: operator tag ${next_operator_tag} already exists at ${existing_operator_tag_target}"
			continue
		fi

		log "Operator tag ${next_operator_tag} already exists at HEAD; scheduling chart publish only"
	else
		create_or_push_tag "${next_operator_tag}" "${head_sha}" "Automated dependency patch release ${next_operator_tag}"
		add_operator_release "$next_operator_version" "$next_operator_tag" "$next_operator_tag"
	fi

	create_or_push_tag "${next_chart_tag}" "${head_sha}" "Automated chart patch release ${next_chart_tag} for ${next_operator_tag}"
	add_chart_release "$next_chart_version" "$next_operator_version" "$next_chart_tag" "$next_chart_tag"
done
