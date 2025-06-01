#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
namespace="snapshot-storage"

cd ${armada_perf_dir}/persistent-storage

declare -A blockVolumeSizes=(['2G']="1" ['20G']="1 10" ['200G']="1 10 100" ['2000G']="1 10 100 1000")
mapfile -d '' sortedBlockVolumeSizes < <(printf '%s\0' "${!blockVolumeSizes[@]}" | sort -rz)

for bvs in ${sortedBlockVolumeSizes[@]}; do
    printf "\n%s - Running tests with %sB block storage volume\n" "$(date +%T)" "${bvs}"

    for bvds in ${blockVolumeSizes[$bvs]}; do
        printf "\n%s - Running test with %sGB data\n" "$(date +%T)" "${bvds}"

        units="GB"
        eachFileSiize=$((bvds / 10))
        if [[ ${eachFileSiize} -eq 0 ]]; then
            units="MB"
            eachFileSiize=$((bvds * 100))
        fi
        # Setup the block storage volume
        helm install perf-snapshot-setup ./imageDeploy/snapshot-storage --namespace=${namespace} --set action=setup --set fileCount=10 --set fileSize=${eachFileSiize} --set fileSizeUnits=${units} --set pvc.storageSize="${bvs}" --wait --timeout 10m

        # Create block storage volume snapshot
        helm install perf-snapshot-backup ./imageDeploy/snapshot-storage --namespace=${namespace} --set action=backup
        sleep 10
        kubectl wait -n ${namespace} --for=jsonpath='{.status.readyToUse}'=true VolumeSnapshot/snapshot-csi-block-perf-pvc --timeout 10m

        # Create new block storage volume from snapshot
        helm install perf-snapshot-restore ./imageDeploy/snapshot-storage --namespace=${namespace} --set action=restore --set pvc.storageSize="${bvs}" --wait --timeout 10m

        helm uninstall perf-snapshot-restore --namespace=${namespace}
        sleep 10
        helm uninstall perf-snapshot-backup --namespace=${namespace}
        sleep 10
        helm uninstall perf-snapshot-setup --namespace=${namespace}

        printf "\n%s -Waiting for cleanup\n" "$(date +%T)"
        kubectl wait --namespace=${namespace} --for=delete deployment/perf-snapshot-setup --timeout 15m
        kubectl wait --namespace=${namespace} --for=delete pvc/perf-pvc-cos --timeout 15m
        kubectl wait --namespace=${namespace} --for=delete VolumeSnapshot/snapshot-csi-block-perf-pvc --timeout 15m
        sleep 10
    done
    echo
done
