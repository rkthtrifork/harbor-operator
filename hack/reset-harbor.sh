#!/bin/bash
# This script deletes all Harbor projects and registries.
# It uses basic authentication with username "admin" and password "Harbor12345"
# and communicates with the Harbor API at the base URL below.
#
# Requirements:
# - curl
# - jq (for JSON parsing)
#
# Note: This script paginates through the results with a page size of 100.
# Adjust the page size if necessary.

# Check if jq is installed
if ! command -v jq &>/dev/null; then
    echo "Error: jq is not installed. Please install jq to proceed."
    exit 1
fi

# Base URL and credentials
BASE_URL="https://core.harbor.domain"
USERNAME="admin"
PASSWORD="Harbor12345"

echo "Starting deletion of all Harbor projects and registries..."

# --- Delete Projects ---
echo "Fetching Harbor projects..."
page=1
while true; do
    response=$(curl -k -s -u "${USERNAME}:${PASSWORD}" "${BASE_URL}/api/v2.0/projects?page=${page}&page_size=100")
    project_count=$(echo "$response" | jq 'length' 2>/dev/null)
    # Default to 0 if project_count is empty or null
    if [ -z "$project_count" ] || [ "$project_count" = "null" ]; then
        project_count=0
    fi

    # Break the loop if no projects are returned
    if [ "$project_count" -eq 0 ]; then
        break
    fi

    echo "Processing page ${page} with ${project_count} project(s)..."
    for project_id in $(echo "$response" | jq -r '.[].project_id'); do
        echo "Deleting project with ID: ${project_id}"
        delete_response=$(curl -k -s -u "${USERNAME}:${PASSWORD}" -X DELETE "${BASE_URL}/api/v2.0/projects/${project_id}")
        echo "Response: ${delete_response}"
    done

    page=$((page + 1))
done

# --- Delete Registries ---
echo "Fetching Harbor registries..."
page=1
while true; do
    response=$(curl -k -s -u "${USERNAME}:${PASSWORD}" "${BASE_URL}/api/v2.0/registries?page=${page}&page_size=100")
    registry_count=$(echo "$response" | jq 'length' 2>/dev/null)
    # Default to 0 if registry_count is empty or null
    if [ -z "$registry_count" ] || [ "$registry_count" = "null" ]; then
        registry_count=0
    fi

    # Break the loop if no registries are returned
    if [ "$registry_count" -eq 0 ]; then
        break
    fi

    echo "Processing page ${page} with ${registry_count} registry(s)..."
    for registry_id in $(echo "$response" | jq -r '.[].id'); do
        echo "Deleting registry with ID: ${registry_id}"
        delete_response=$(curl -k -s -u "${USERNAME}:${PASSWORD}" -X DELETE "${BASE_URL}/api/v2.0/registries/${registry_id}")
        echo "Response: ${delete_response}"
    done

    page=$((page + 1))
done

echo "Deletion of all projects and registries is complete."
