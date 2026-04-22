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

echo "Running gemini-cli for agent ${AGENT_NAME}..."
gemini-cli --yolo "$(cat $PROMPT_FILE)"