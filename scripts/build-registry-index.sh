#!/usr/bin/env bash
# build-registry-index.sh
#
# Scans all versioned sensor manifest files (*-v*.json) in the current
# directory and generates a consolidated index.json for the registry.
#
# This script is called by the publish-sensor-registry CI workflow
# after writing a new versioned manifest to the registry-pages branch.
#
# Requirements: jq, sort (with -V for version sorting)
#
# Output format:
# {
#   "generated": "2025-06-15T12:00:00Z",
#   "sensors": [
#     {
#       "id": "hw-sensor-file-canary",
#       "name": "File Canary (FIM)",
#       "category": "file",
#       "icon_svg": "...",
#       "latest": "1.2.0",
#       "versions": [
#         { "v": "1.0.0", "min_hub_api": "1" }
#       ]
#     }
#   ]
# }

set -euo pipefail

# Collect all versioned manifest files
shopt -s nullglob
MANIFEST_FILES=(*-v*.json)

if [ ${#MANIFEST_FILES[@]} -eq 0 ]; then
  echo "⚠️  No versioned manifest files found. Writing empty index."
  NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  jq -n --arg gen "$NOW" '{ generated: $gen, sensors: [] }' > index.json
  exit 0
fi

echo "📊 Building index from ${#MANIFEST_FILES[@]} manifest file(s)..."

# Step 1: Extract metadata from each manifest into a flat JSONL stream
ENTRIES=""
for FILE in "${MANIFEST_FILES[@]}"; do
  # Validate the file has required fields for the registry
  if ! jq -e '.id and .name and .version and .category' "$FILE" >/dev/null 2>&1; then
    echo "⚠️  Skipping invalid or incomplete manifest: $FILE"
    continue
  fi

  # Skip deprecated versions
  if jq -e '.deprecated == true' "$FILE" >/dev/null 2>&1; then
    echo "⏭️  Skipping deprecated version: $FILE"
    continue
  fi

  ENTRY=$(jq -c '{
    id: .id,
    name: .name,
    category: .category,
    icon_svg: .icon_svg,
    version: .version
  }' "$FILE")

  if [ -n "$ENTRY" ]; then
    ENTRIES="${ENTRIES}${ENTRY}\n"
  fi
done

if [ -z "$ENTRIES" ]; then
  echo "⚠️  No valid manifests found. Writing empty index."
  NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  jq -n --arg gen "$NOW" '{ generated: $gen, sensors: [] }' > index.json
  exit 0
fi

# Step 2: Group by sensor ID, collect versions, determine latest
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo -e "$ENTRIES" | jq -s '
  # Group by sensor ID
  group_by(.id) |
  map(
    # Sort versions within each group (lexicographic works for semver with same digit count)
    sort_by(.version | split(".") | map(tonumber)) |
    {
      id: .[0].id,
      name: .[0].name,
      category: .[0].category,
      icon_svg: .[0].icon_svg,
      latest: .[-1].version,
      versions: [.[] | { v: .version }]
    }
  ) |
  sort_by(.id)
' | jq --arg gen "$NOW" '{ generated: $gen, sensors: . }' > index.json

# Step 3: Report
SENSOR_COUNT=$(jq '.sensors | length' index.json)
VERSION_COUNT=$(jq '[.sensors[].versions[]] | length' index.json)
echo "✅ index.json generated:"
echo "   Sensors:  $SENSOR_COUNT"
echo "   Versions: $VERSION_COUNT"
echo "   Generated: $NOW"
