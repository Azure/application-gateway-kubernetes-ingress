#!/bin/bash

set -euo pipefail

# This script is used to replicate the helm chart from storage account to OCI registry
# Note: This script is being added for historical purpose. The helm chart is now being pushed to
# the OCI registry directly from the pipeline
# Usage: ./replicate-chart.sh <target_acr> <chart_version>

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <target_acr> <chart_version>"
    exit 1
fi

DESTINATION_ACR="$1"
DESTINATION_REPO="$DESTINATION_ACR/public/azure-application-gateway/charts"
SOURCE_VERSION="$2"

SOURCE_CHART_BASE_URL="https://appgwingress.blob.core.windows.net/ingress-azure-helm-package"
SOURCE_CHART_NAME="ingress-azure"
CHART_TAR="$SOURCE_CHART_NAME-$SOURCE_VERSION".tgz
SOURCE_URL="$SOURCE_CHART_BASE_URL/$CHART_TAR"

wget "$SOURCE_URL"

helm push "$CHART_TAR" oci://"$DESTINATION_REPO"