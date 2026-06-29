#!/bin/sh
# Forge concept: issue-list (Jira Cloud via REST API v3)
# Input: CODEV_JIRA_PROJECT (optional, e.g. "PROJ")
# Output: JSON [{number, title, url, labels, createdAt, author, assignees}]
set -e

if [ -z "$JIRA_BASE_URL" ]; then
  echo "JIRA_BASE_URL is not set (e.g. https://team.atlassian.net)" >&2
  exit 1
fi

if [ -z "$JIRA_USER_EMAIL" ] || [ -z "$JIRA_API_TOKEN" ]; then
  echo "JIRA_USER_EMAIL and JIRA_API_TOKEN must be set" >&2
  exit 1
fi

JQL="statusCategory != Done ORDER BY updated DESC"
if [ -n "$CODEV_JIRA_PROJECT" ]; then
  JQL="project = ${CODEV_JIRA_PROJECT} AND statusCategory != Done ORDER BY updated DESC"
fi

curl -sf -u "${JIRA_USER_EMAIL}:${JIRA_API_TOKEN}" \
  -H "Accept: application/json" \
  -G "${JIRA_BASE_URL}/rest/api/3/search" \
  --data-urlencode "jql=${JQL}" \
  --data-urlencode "fields=summary,status,labels,created,creator,assignee,components" \
  --data-urlencode "maxResults=200" \
  | jq --arg base "$JIRA_BASE_URL" '[.issues[] | {
    number: .key,
    title: .fields.summary,
    url: ($base + "/browse/" + .key),
    labels: [(.fields.labels // [])[] | { name: . }] + [(.fields.components // [])[] | { name: .name }],
    createdAt: .fields.created,
    author: (if .fields.creator then { login: .fields.creator.displayName } else null end),
    assignees: (if .fields.assignee then [{ login: .fields.assignee.displayName }] else [] end)
  }]'
