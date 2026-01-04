#!/bin/bash

# Build, push, and deploy eratemanager
# Usage: ./build-and-deploy.sh [VERSION] [NAMESPACE]
# Example: ./build-and-deploy.sh v0.2.0 eratemanager

set -e

# Configuration
VERSION="${1:-latest}"
NAMESPACE="${2:-eratemanager}"
REGISTRY="ghcr.io"
IMAGE_NAME="bher20/eratemanager"
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${VERSION}"
DEPLOYMENT_NAME="eratemanager"

echo "=========================================="
echo "Building and Deploying eRateManager"
echo "=========================================="
echo "Version: ${VERSION}"
echo "Namespace: ${NAMESPACE}"
echo "Image: ${FULL_IMAGE}"
echo ""

# Step 1: Build Docker image
echo "[1/4] Building Docker image..."
docker build --build-arg VERSION="${VERSION}" -t "${FULL_IMAGE}" -f Containerfile .
if [ $? -ne 0 ]; then
    echo "❌ Docker build failed"
    exit 1
fi
echo "✅ Docker image built successfully"
echo ""

# Step 2: Push to registry
echo "[2/4] Pushing image to registry..."
docker push "${FULL_IMAGE}"
if [ $? -ne 0 ]; then
    echo "❌ Docker push failed"
    exit 1
fi
echo "✅ Image pushed successfully"
echo ""

# Step 3: Update deployment
echo "[3/5] Updating Kubernetes deployment..."
kubectl set image deployment/${DEPLOYMENT_NAME} \
    ${DEPLOYMENT_NAME}="${FULL_IMAGE}" \
    -n ${NAMESPACE}
if [ $? -ne 0 ]; then
    echo "❌ Kubectl set image failed"
    exit 1
fi
echo "✅ Deployment updated"
echo ""

# Step 4: Force rollout (even if image tag is unchanged)
echo "[4/5] Forcing rollout restart..."
kubectl rollout restart deployment/${DEPLOYMENT_NAME} -n ${NAMESPACE}
if [ $? -ne 0 ]; then
    echo "❌ Rollout restart failed"
    exit 1
fi
echo "✅ Rollout restart triggered"
echo ""

# Step 5: Wait for rollout
echo "[5/5] Waiting for rollout to complete..."
kubectl rollout status deployment/${DEPLOYMENT_NAME} \
    -n ${NAMESPACE} \
    --timeout=5m
if [ $? -ne 0 ]; then
    echo "❌ Rollout failed or timed out"
    exit 1
fi
echo "✅ Rollout completed successfully"
echo ""

echo "=========================================="
echo "✅ Deployment complete!"
echo "=========================================="
echo "Image: ${FULL_IMAGE}"
echo "Namespace: ${NAMESPACE}"
echo ""
