#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

cluster_prefix=$1
password=$2 # pragma: allowlist secret
num_clusters=$3

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
k8s_apiserver_perf_dir=${armada_perf_dir}/k8s-apiserver
jmeter_config="${k8s_apiserver_perf_dir}/jmeter-config"

# If the number of clusters wasn't specified, let's use them all
if [[ -z ${num_clusters} ]]; then
  clusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json | jq -r --arg cnp "${cluster_prefix}" '.[] | select(.name | startswith($cnp)) | .name')
  num_clusters=$(echo ${clusters} | wc -l)
else
  clusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json | jq -r --arg cnp "${cluster_prefix}" '.[] | select(.name | startswith($cnp)) | .name' | head -${num_clusters})
  actual_clusters=$(echo ${clusters} | wc -l)
  if [[ ${actual_clusters} -ne ${num_clusters} ]]; then
    echo "ERROR: Only ${actual_clusters} clusters when ${num_clusters} required"
    exit 1
  fi
fi

# Ensure JMeter config is created and empty
if [[ -d ${jmeter_config} ]]; then
  rm -r ${jmeter_config}
fi
mkdir -p ${jmeter_config}

cp ${k8s_apiserver_perf_dir}/requests.csv ${jmeter_config}/.
cp ${k8s_apiserver_perf_dir}/*.jmx ${jmeter_config}/.

# For each cluster
for cluster_name in $(echo ${clusters}); do
  cluster_config_dir="${perf_dir}"/config/"${cluster_name}"

  # Ensure cluster config is available for this cluster
  if [ ! -d "${cluster_config_dir}" ]; then
    mkdir -p "${cluster_config_dir}"
    ${perf_dir}/bin/armada-perf-client2 cluster config --cluster "${cluster_name}" --admin
    mv kubeConfigAdmin-"${cluster_name}".zip "${cluster_config_dir}"
    unzip -o -j -d "${cluster_config_dir}" "${cluster_config_dir}"/kubeConfigAdmin-"${cluster_name}".zip
  fi

  conf_yaml=$(ls ${cluster_config_dir}/*.yml)
  export KUBECONFIG=${conf_yaml}

  cat ${cluster_config_dir}/admin-key.pem ${cluster_config_dir}/admin.pem | openssl pkcs12 -export -out ${cluster_config_dir}/cert.p12 -name ${cluster_name} -passout pass:${password} # pragma: allowlist secret

  keytool -importkeystore -noprompt -srckeystore ${cluster_config_dir}/cert.p12 -srcstoretype pkcs12 -destkeystore ${jmeter_config}/cert.jks -destalias ${cluster_name} -srcstorepass ${password} -deststorepass ${password} -srcalias ${cluster_name} -deststoretype pkcs12

  # Get the api server information from the kube config
  apiserver=$(kubectl config view -o json | jq --raw-output '. | .clusters[].cluster.server')
  IFS=':' read -r -a server_array <<<"${apiserver}"
  protocol=${server_array[0]}
  hostname=${server_array[1]:3-1}
  port=${server_array[2]}

  # java keystore aliases (JKS) are always all lowercase
  alias_name="${cluster_name,,}"
  echo "${alias_name},${protocol},${hostname},${port}" >>${jmeter_config}/clusters.csv
done
