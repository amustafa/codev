#!/bin/sh
# Forge concept: issue-view (Jira Cloud via REST API v3)
# Input: CODEV_ISSUE_ID (e.g. "PROJ-123")
# Output: JSON {title, body, state, comments[]}
set -e

if [ -z "$JIRA_BASE_URL" ]; then
  echo "JIRA_BASE_URL is not set (e.g. https://team.atlassian.net)" >&2
  exit 1
fi

if [ -z "$JIRA_USER_EMAIL" ] || [ -z "$JIRA_API_TOKEN" ]; then
  echo "JIRA_USER_EMAIL and JIRA_API_TOKEN must be set" >&2
  exit 1
fi

if [ -z "$CODEV_ISSUE_ID" ]; then
  echo "CODEV_ISSUE_ID is not set" >&2
  exit 1
fi

# Fetch issue with comment expansion
curl -sf -u "${JIRA_USER_EMAIL}:${JIRA_API_TOKEN}" \
  -H "Accept: application/json" \
  "${JIRA_BASE_URL}/rest/api/3/issue/${CODEV_ISSUE_ID}?fields=summary,description,status,comment" \
  | jq '{
    title: .fields.summary,
    body: (
      if .fields.description then
        [.fields.description | recurse(.content[]?) | select(.type == "text") | .text] | join("")
      else
        ""
      end
    ),
    state: .fields.status.name,
    comments: [
      (.fields.comment.comments // [])[] | {
        body: (
          if .body then
            [.body | recurse(.content[]?) | select(.type == "text") | .text] | join("")
          else
            ""
          end
        ),
        createdAt: .created,
        author: { login: .author.displayName }
      }
    ]
  }'
