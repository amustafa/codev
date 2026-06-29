#!/bin/sh
# Forge concept: issue-search (Jira Cloud via REST API v3)
# Input: CODEV_ISSUE_STATE (optional: open|closed|all, default: open)
#        CODEV_JIRA_PROJECT (optional, e.g. "PROJ")
#        CODEV_SEARCH_QUERY (optional, text search)
# Output: JSON [{number, title, url, labels, createdAt, author, assignees, body}]
set -e

if [ -z "$JIRA_BASE_URL" ]; then
  echo "JIRA_BASE_URL is not set (e.g. https://team.atlassian.net)" >&2
  exit 1
fi

if [ -z "$JIRA_USER_EMAIL" ] || [ -z "$JIRA_API_TOKEN" ]; then
  echo "JIRA_USER_EMAIL and JIRA_API_TOKEN must be set" >&2
  exit 1
fi

case "${CODEV_ISSUE_STATE:-open}" in
  closed) STATE_FILTER="statusCategory = Done" ;;
  all)    STATE_FILTER="" ;;
  *)      STATE_FILTER="statusCategory != Done" ;;
esac

JQL_PARTS=""
if [ -n "$CODEV_JIRA_PROJECT" ]; then
  JQL_PARTS="project = ${CODEV_JIRA_PROJECT}"
fi
if [ -n "$STATE_FILTER" ]; then
  if [ -n "$JQL_PARTS" ]; then
    JQL_PARTS="${JQL_PARTS} AND ${STATE_FILTER}"
  else
    JQL_PARTS="${STATE_FILTER}"
  fi
fi
if [ -n "$CODEV_SEARCH_QUERY" ]; then
  if [ -n "$JQL_PARTS" ]; then
    JQL_PARTS="${JQL_PARTS} AND text ~ \"${CODEV_SEARCH_QUERY}\""
  else
    JQL_PARTS="text ~ \"${CODEV_SEARCH_QUERY}\""
  fi
fi

JQL="${JQL_PARTS:-"ORDER BY updated DESC"}"
if [ -n "$JQL_PARTS" ]; then
  JQL="${JQL_PARTS} ORDER BY updated DESC"
fi

curl -sf -u "${JIRA_USER_EMAIL}:${JIRA_API_TOKEN}" \
  -H "Accept: application/json" \
  -G "${JIRA_BASE_URL}/rest/api/3/search" \
  --data-urlencode "jql=${JQL}" \
  --data-urlencode "fields=summary,description,status,labels,created,creator,assignee,components" \
  --data-urlencode "maxResults=200" \
  | jq --arg base "$JIRA_BASE_URL" '[.issues[] | {
    number: .key,
    title: .fields.summary,
    url: ($base + "/browse/" + .key),
    labels: [(.fields.labels // [])[] | { name: . }] + [(.fields.components // [])[] | { name: .name }],
    createdAt: .fields.created,
    author: (if .fields.creator then { login: .fields.creator.displayName } else null end),
    assignees: (if .fields.assignee then [{ login: .fields.assignee.displayName }] else [] end),
    body: (
      if .fields.description then
        [.fields.description | recurse(.content[]?) | select(.type == "text") | .text] | join("")
      else
        ""
      end
    )
  }]'
