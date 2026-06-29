#!/bin/sh
# Forge concept: recently-closed (Jira Cloud via REST API v3)
# Input: CODEV_JIRA_PROJECT (optional), CODEV_SINCE_DATE (optional, ISO date)
# Output: JSON [{number, title, url, labels, createdAt, closedAt}]
set -e

if [ -z "$JIRA_BASE_URL" ]; then
  echo "JIRA_BASE_URL is not set (e.g. https://team.atlassian.net)" >&2
  exit 1
fi

if [ -z "$JIRA_USER_EMAIL" ] || [ -z "$JIRA_API_TOKEN" ]; then
  echo "JIRA_USER_EMAIL and JIRA_API_TOKEN must be set" >&2
  exit 1
fi

SINCE="${CODEV_SINCE_DATE:--7d}"
JQL="statusCategory = Done AND status changed AFTER \"${SINCE}\""
if [ -n "$CODEV_JIRA_PROJECT" ]; then
  JQL="project = ${CODEV_JIRA_PROJECT} AND ${JQL}"
fi
JQL="${JQL} ORDER BY updated DESC"

curl -sf -u "${JIRA_USER_EMAIL}:${JIRA_API_TOKEN}" \
  -H "Accept: application/json" \
  -G "${JIRA_BASE_URL}/rest/api/3/search" \
  --data-urlencode "jql=${JQL}" \
  --data-urlencode "fields=summary,status,labels,created,resolutiondate,components" \
  --data-urlencode "maxResults=200" \
  | jq --arg base "$JIRA_BASE_URL" '[.issues[] | {
    number: .key,
    title: .fields.summary,
    url: ($base + "/browse/" + .key),
    labels: [(.fields.labels // [])[] | { name: . }] + [(.fields.components // [])[] | { name: .name }],
    createdAt: .fields.created,
    closedAt: .fields.resolutiondate
  }]'
