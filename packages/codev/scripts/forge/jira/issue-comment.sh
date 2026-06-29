#!/bin/sh
# Forge concept: issue-comment (Jira Cloud via REST API v3)
# Input: CODEV_ISSUE_ID (e.g. "PROJ-123"), CODEV_COMMENT_BODY
# Output: exit code only
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

# Jira v3 comment body uses ADF format; wrap plain text in a paragraph node
curl -sf -u "${JIRA_USER_EMAIL}:${JIRA_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -X POST "${JIRA_BASE_URL}/rest/api/3/issue/${CODEV_ISSUE_ID}/comment" \
  -d "$(jq -n --arg body "$CODEV_COMMENT_BODY" '{
    body: {
      type: "doc",
      version: 1,
      content: [{
        type: "paragraph",
        content: [{
          type: "text",
          text: $body
        }]
      }]
    }
  }')" \
  -o /dev/null
