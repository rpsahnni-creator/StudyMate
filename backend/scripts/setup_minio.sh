#!/bin/bash
# Start MinIO for local dev with a 24-hour lifecycle on the temp bucket.
set -euo pipefail

CONTAINER_NAME="studyapp-minio"
BUCKET="studyapp-temp"
MINIO_USER="${MINIO_ROOT_USER:-minioadmin}"
MINIO_PASS="${MINIO_ROOT_PASSWORD:-minioadmin123}"

echo "Starting MinIO container..."
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
docker run -d \
  --name "$CONTAINER_NAME" \
  -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER="$MINIO_USER" \
  -e MINIO_ROOT_PASSWORD="$MINIO_PASS" \
  minio/minio server /data --console-address ":9001"

echo "Waiting for MinIO to become ready..."
for i in $(seq 1 30); do
  if curl -sf "http://localhost:9000/minio/health/live" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "Configuring mc alias and bucket..."
docker run --rm --network container:"$CONTAINER_NAME" \
  minio/mc alias set local http://localhost:9000 "$MINIO_USER" "$MINIO_PASS"
docker run --rm --network container:"$CONTAINER_NAME" \
  minio/mc mb --ignore-existing "local/$BUCKET"

# Lifecycle: expire objects under temp/ after 1 day.
LIFECYCLE_JSON=$(cat <<EOF
{
  "Rules": [
    {
      "ID": "expire-temp-24h",
      "Status": "Enabled",
      "Filter": { "Prefix": "temp/" },
      "Expiration": { "Days": 1 }
    }
  ]
}
EOF
)
echo "$LIFECYCLE_JSON" | docker run --rm -i --network container:"$CONTAINER_NAME" \
  minio/mc ilm import "local/$BUCKET" || echo "Note: lifecycle import may require mc 2023+; configure via console if this fails."

echo ""
echo "MinIO is running."
echo "  API:     http://localhost:9000"
echo "  Console: http://localhost:9001"
echo "  Bucket:  $BUCKET (temp/ prefix expires after 24h)"
echo ""
echo "Add to backend .env:"
echo "  STORAGE_ENDPOINT=http://localhost:9000"
echo "  STORAGE_BUCKET=$BUCKET"
echo "  STORAGE_REGION=us-east-1"
echo "  STORAGE_ACCESS_KEY=$MINIO_USER"
echo "  STORAGE_SECRET_KEY=$MINIO_PASS"
echo "  STORAGE_USE_SSL=false"
