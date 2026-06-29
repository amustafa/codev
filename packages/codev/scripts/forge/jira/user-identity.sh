#!/bin/sh
# Forge concept: user-identity (Jira Cloud via REST API v3)
# Output: plain text display name
set -e

if [ -z "$JIRA_BASE_URL" ]; then
  echo "JIRA_BASE_URL is not set (e.g. https://team.atlassian.net)" >&2
  exit 1
fi

if [ -z "$JIRA_USER_EMAIL" ] || [ -z "$JIRA_API_TOKEN" ]; then
  echo "JIRA_USER_EMAIL and JIRA_API_TOKEN must be set" >&2
  exit 1
fi

curl -sf -u "${JIRA_USER_EMAIL}:${JIRA_API_TOKEN}" \
  -H "Accept: application/json" \
  "${JIRA_BASE_URL}/rest/api/3/myself" \
  | jq -r '.displayName'
