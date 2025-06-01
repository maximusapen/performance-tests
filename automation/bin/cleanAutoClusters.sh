#!/bin/bash -e

perf_dir="/performance"
export GOPATH=${perf_dir}
export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}

echo "cleanAutoClusters.sh parameters: $1 $2 $3"

if [[ -z "$1" ]]; then
	# Default is to delete all Perf clusters
	cluster_prefix_list="Perf"
else
	echo Requested to delete clusters with cluster prefixes: $1
	cluster_prefix_list=$(echo $1 | sed "s/,/ /g")
fi

excludeClusterPrefix=""
if [[ $2 == "--excludeClusterPrefix" ]]; then
	excludeClusterPrefix=$3
fi

echo cluster_prefix_list: $cluster_prefix_list

# Also delete any hollow node clusters
custom_spawn_cluster="hollow_node_pods_cluster_auto"

# List all clusters
echo Listing all clusters on the carrier
"${perf_dir}"/bin/armada-perf-client2 cluster ls --json | grep name | grep -v ingressHostname

for cluster_prefix in ${cluster_prefix_list}; do
	# Get all the clusters for each cluster_prefix

	cluster_list=$("${perf_dir}"/bin/armada-perf-client2 cluster ls --json | grep name | grep -v ingressHostname | grep "${cluster_prefix}" | awk '{print $2}' | sed "s/,/ /g" | sed "s/\"//g")
	echo "For $cluster_prefix, cluster_list:"
	echo "$cluster_list"

	for cluster in ${cluster_list}; do
		if [[ "${cluster}" == "${excludeClusterPrefix}1" || "${cluster}" == "${excludeClusterPrefix}-load1" ]]; then
			# Exclude deleting current test clusters and its load driver
			echo "Excluding ${cluster} for delete"
		else
			echo "Deleting ${cluster}"
			printf "\n%s - Deleting %s and associated config files\n" "$(date +%T)" "${cluster}"
			"${perf_dir}"/bin/armada-perf-client2 cluster rm --cluster "${cluster}" --force-delete-storage

			if [[ -d "${perf_dir}/config/${cluster}" ]]; then

				# Esnure we have no leftover persistent volumes
				cluster_config_dir="${perf_dir}"/config/"${cluster}"

				if [[ -d "${cluster_config_dir}"/persistent-storage ]]; then
					for pv in "${cluster_config_dir}"/persistent-storage/*; do
						fn=${pv##*/}
						IFS='-' read -r -a vol_arr <<<"${fn}"
						voltyp="${vol_arr[1]}"
						volid="${vol_arr[-1]}"

						sl_data=$(ibmcloud sl "${voltyp}" volume-list | grep ${volid})
						if [[ -n "${sl_data}" ]]; then
							IFS=' ' read -r -a vol_data <<<"${sl_data}"

							# Double check that we've got the right volume
							if [[ "${vol_data[0]}" == "${volid}" ]]; then
								# Check there are no active transactions before trying to cancel
								if [[ ${vol_data[7]} == "0" ]]; then
									ibmcloud sl "${voltyp}" volume-cancel "${volid}" --force
								fi
							fi
						fi
					done
				fi
				rm -rf "${cluster_config_dir}"
			else
				echo Config directory "${perf_dir}/config/${cluster}" not found.
			fi

			if [[ $cluster == *"KMarkAuto"* ]]; then
				echo This is a hollow node testing cluster, checking to delete hollow node cluster
				# Check and delete hollow node clusters if found
				echo Checking for hollow node cluster
				hollow_node_list=$("${perf_dir}"/bin/armada-perf-client2 cluster ls --json | grep name | grep -v ingressHostname | grep "${custom_spawn_cluster}" | awk '{print $2}' | sed "s/,/ /g" | sed "s/\"//g")
				echo hollow_node_list: $hollow_node_list

				for hollow_node_cluster in ${hollow_node_list}; do
					echo Deleting $hollow_node_cluster
					"${perf_dir}"/bin/armada-perf-client2 cluster rm --cluster "${hollow_node_cluster}" --force-delete-storage
				done
			fi
		fi
	done

done
