#!/bin/bash
set -e

if [ -z "$AGENT_NAME" ]; then
  echo "AGENT_NAME environment variable is not set."
  exit 1
fi

echo "Cloning https://github.com/ai-on-gke/ai-factory.git..."
git clone https://github.com/ai-on-gke/ai-factory.git
cd ai-factory

PROMPT_FILE=".agents/${AGENT_NAME}/agent.md"
if [ ! -f "$PROMPT_FILE" ]; then
  echo "Prompt file $PROMPT_FILE not found."
  exit 1
fi

export GEMINI_API_KEY="fake"
export GEMINI_CLI_TRUST_WORKSPACE=true

if [ -n "$GEMINI_SERVICE_PORTAL" ]; then
  export HTTPS_PROXY="$GEMINI_SERVICE_PORTAL"
fi
if [ -n "$GEMINI_SERVICE_PORTAL_CA_CERTS" ]; then
  export NODE_EXTRA_CA_CERTS="$GEMINI_SERVICE_PORTAL_CA_CERTS"
fi

echo "Running gemini for agent ${AGENT_NAME}..."
cat "${PROMPT_FILE}" | gemini --yolo