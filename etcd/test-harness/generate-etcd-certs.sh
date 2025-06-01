# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Create certificates for etcd cluster. If <directory> isn't specified then the certificates
# will be stored in kubernetes secrets.
# Usage: ./generate-etcd-certs.sh [<directory>]
# NOTE: This must be run with armada namespace. See "find etcd endpoints to sign certs with" below to understand why.

. etcd-perftest-config

OUTPUT_DIR=""
if [[ $# -eq 1 ]]; then
    OUTPUT_DIR=$1
fi

if [[ ! -x cfssl ]]; then
    echo "Retrieving cfssl and cfssljson"
    VERSION=$(curl --silent "https://api.github.com/repos/cloudflare/cfssl/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    VNUMBER=${VERSION#"v"}
    OS_NAME=$(uname -s)
    if [[ ${OS_NAME} == 'Darwin' ]]; then
        curl -L https://github.com/cloudflare/cfssl/releases/download/${VERSION}/cfssl_${VNUMBER}_darwin_amd64 > cfssl
        curl -L https://github.com/cloudflare/cfssl/releases/download/${VERSION}/cfssljson_${VNUMBER}_darwin_amd64 > cfssljson
    else
        wget https://github.com/cloudflare/cfssl/releases/download/${VERSION}/cfssl_${VNUMBER}_linux_amd64 -O cfssl
        wget https://github.com/cloudflare/cfssl/releases/download/${VERSION}/cfssljson_${VNUMBER}_linux_amd64 -O cfssljson
    fi
    chmod +x cfssl
    chmod +x cfssljson

    #sudo mv cfssl /usr/local/bin
    #sudo mv cfssljson /usr/local/bin
fi

DIR=$(pwd)
CFSSL=${DIR}/cfssl
CFSSLJSON=${DIR}/cfssljson
${CFSSL} version

if [[ -d cfssl_cert_directory ]]; then
    rm -rf cfssl_cert_directory
fi
mkdir cfssl_cert_directory

echo "find etcd endpoints to sign certs with"
etcd_endpoints_operator=$(kubectl -n ${NAMESPACE} get cm armada-etcd-configmap -o jsonpath='{.data.ETCD_ENDPOINTS_OPERATOR}')

echo "convert etcd_endpoints_operator to a wildcard"
etcd_endpoints_operator_wildcard=$(echo "${etcd_endpoints_operator}" | awk -Fetcd-.*-1 '{print $2}' | awk -F: '{print $1}' | xargs -I{} echo "*{}")
echo "wildcard: " ${etcd_endpoints_operator_wildcard}

echo "Generate CA cert and pem"
pushd cfssl_cert_directory
${CFSSL} gencert -initca ${DIR}/ca-csr.json | ${CFSSLJSON} -bare ca
popd

echo "Apply CA secrets"
if [[ -z ${OUTPUT_DIR} ]]; then
    kubectl create secret generic -n ${NAMESPACE} ${ETCDCLUSTER_NAME}-ca-tls --from-file cfssl_cert_directory/ca-key.pem --from-file cfssl_cert_directory/ca.pem
else
    kubectl create secret generic -n ${NAMESPACE} ${ETCDCLUSTER_NAME}-ca-tls --from-file cfssl_cert_directory/ca-key.pem --from-file cfssl_cert_directory/ca.pem --dry-run=client -o yaml > ${OUTPUT_DIR}/${ETCDCLUSTER_NAME}-ca-tls.yaml
fi

echo "template the cfssl json file"
sed -e "s/ETCD_ENDPOINTS_OPERATOR_WILDCARD/${etcd_endpoints_operator_wildcard}/g" -e "s/ETCDCLUSTER_NAME/${ETCDCLUSTER_NAME}/g" etcd-peer-csr.json.template > cfssl_cert_directory/etcd-peer-csr.json
sed -e "s/ETCD_ENDPOINTS_OPERATOR_WILDCARD/${etcd_endpoints_operator_wildcard}/g" -e "s/ETCDCLUSTER_NAME/${ETCDCLUSTER_NAME}/g" etcd-server-csr.json.template > cfssl_cert_directory/etcd-server-csr.json

echo "Generate peer certs"
pushd cfssl_cert_directory
cat "etcd-peer-csr.json" | ${CFSSL} gencert -ca="ca.pem" -ca-key="ca-key.pem" -config="${DIR}/ca-profiles.json" -profile=peer - | ${CFSSLJSON} -bare etcd-peer

echo "Generate client certs"
cat ${DIR}/etcd-client-csr.json | ${CFSSL} gencert -ca="ca.pem" -ca-key="ca-key.pem" -config="${DIR}/ca-profiles.json" -profile=client - | ${CFSSLJSON} -bare etcd-client

echo "Generate server certs"
cat "etcd-server-csr.json" | ${CFSSL} gencert -ca="ca.pem" -ca-key="ca-key.pem" -config="${DIR}/ca-profiles.json" -profile=server - | ${CFSSLJSON} -bare etcd-server

echo "Move Regenerated Etcd Cert to the proper names"
cp etcd-peer.pem peer.crt
cp etcd-peer-key.pem peer.key
cp ca.pem peer-ca.crt
cp etcd-server.pem server.crt
cp etcd-server-key.pem server.key
cp ca.pem server-ca.crt
cp etcd-client.pem etcd-client.crt
cp etcd-client-key.pem etcd-client.key
cp ca.pem etcd-client-ca.crt
popd

echo "Reapply the client secrets"
if [[ -z ${OUTPUT_DIR} ]]; then
    kubectl create secret generic -n armada ${ETCDCLUSTER_NAME}-client-tls --from-file cfssl_cert_directory/etcd-client.crt --from-file cfssl_cert_directory/etcd-client.key --from-file cfssl_cert_directory/etcd-client-ca.crt --dry-run=client -o yaml | kubectl apply -f -
else
    kubectl create secret generic -n armada ${ETCDCLUSTER_NAME}-client-tls --from-file cfssl_cert_directory/etcd-client.crt --from-file cfssl_cert_directory/etcd-client.key --from-file cfssl_cert_directory/etcd-client-ca.crt --dry-run=client -o yaml  > ${OUTPUT_DIR}/${ETCDCLUSTER_NAME}-client-tls.yaml
fi

echo "Reapply the server secrets"
if [[ -z ${OUTPUT_DIR} ]]; then
    kubectl create secret generic -n armada ${ETCDCLUSTER_NAME}-server-tls --from-file cfssl_cert_directory/server.crt --from-file cfssl_cert_directory/server.key --from-file cfssl_cert_directory/server-ca.crt --dry-run=client -o yaml | kubectl apply -f -
else
    kubectl create secret generic -n armada ${ETCDCLUSTER_NAME}-server-tls --from-file cfssl_cert_directory/server.crt --from-file cfssl_cert_directory/server.key --from-file cfssl_cert_directory/server-ca.crt --dry-run=client -o yaml > ${OUTPUT_DIR}/${ETCDCLUSTER_NAME}-server-tls.yaml
fi

echo "Reapply the peer secrets"
if [[ -z ${OUTPUT_DIR} ]]; then
    kubectl create secret generic -n armada ${ETCDCLUSTER_NAME}-peer-tls --from-file cfssl_cert_directory/peer.crt --from-file cfssl_cert_directory/peer.key --from-file cfssl_cert_directory/peer-ca.crt --dry-run=client -o yaml | kubectl apply -f -
else
    kubectl create secret generic -n armada ${ETCDCLUSTER_NAME}-peer-tls --from-file cfssl_cert_directory/peer.crt --from-file cfssl_cert_directory/peer.key --from-file cfssl_cert_directory/peer-ca.crt --dry-run=client -o yaml > ${OUTPUT_DIR}/${ETCDCLUSTER_NAME}-peer-tls.yaml
fi

# TODO rm -rf cfssl_cert_directory
