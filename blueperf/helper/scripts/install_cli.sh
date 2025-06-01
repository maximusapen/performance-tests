#!/bin/bash

# Checking if ibmcloud is installed
grn=$'\e[1;32m'
end=$'\e[0m'

IBMCLOUD_PATH=$(command -v ibmcloud)

if [[ $? -ne 0 ]]; then
	printf "\n\n${grn}Installing ibmcloud CLI ...${end}\n"

    wget https://ibm.biz/idt-installer > /tmp/idt-installer
    yes | /tmp/idt-installer uninstall
    rm /tmp/idt-installer
fi

# Update CLI
ibmcloud update -f &> /dev/null

# Check if ibmcloud ks is installed
ibmcloud ks &> /dev/null
if [[ $? -ne 0 ]]; then
	printf "\n\n${grn}Installing IBM Cloud Container Service (ibmcloud ks  ) plugin...${end}\n"
	ibmcloud plugin install container-service -r "IBM Cloud"
fi

# Check if ibmcloud cr is installed
ibmcloud cr &> /dev/null
if [[ $? -ne 0 ]]; then
	printf "\n\n${grn}Installing IBM Cloud Container Registry Service (ibmcloud cr) plugin...${end}\n"
	ibmcloud plugin install container-registry -r "IBM Cloud"
fi

# Checking if kubectl is installed
KUBE_PATH=$(command -v kubectl)

if [[ $? -ne 0 ]]; then
	printf "\n\n${grn}Installing Kubernetes CLI (kubectl)...${end}\n"

	if [[ $OSTYPE =~ .*darwin.* ]]; then
		# OS X
		curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/darwin/amd64/kubectl

	elif [[ $OSTYPE =~ .*linux.* ]]; then
		# Linux
		curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
	fi

	chmod +x ./kubectl
	sudo mv ./kubectl /usr/local/bin/kubectl
fi


# Installing jq
JQ_PATH=$(command -v jq)

if [[ $? -ne 0 ]]; then
	printf "\n\n${grn}Installing jq${end}\n"

	if [[ $OSTYPE =~ .*darwin.* ]]; then
		# OS X
		curl -Lo jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-osx-amd64

	elif [[ $OSTYPE =~ .*linux.* ]]; then
		# Linux
		curl -o jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64
	fi

	chmod +x ./jq
	sudo mv ./jq /usr/local/bin/jq
fi

# Installing yaml
YAML_PATH=$(command -v yaml)

if [[ $? -ne 0 ]]; then
	printf "\n\n${grn}Installing YAML${end}\n"

	if [[ $OSTYPE =~ .*darwin.* ]]; then
		# OS X
		curl -LO https://github.com/mikefarah/yaml/releases/download/1.10/yaml_darwin_amd64
		mv yaml_darwin_amd64 yaml

	elif [[ $OSTYPE =~ .*linux.* ]]; then
		# Linux
		curl -o yaml https://github.com/mikefarah/yaml/releases/download/1.8/yaml_linux_amd64
	fi

	chmod +x ./yaml
	sudo mv ./yaml /usr/local/bin/yaml
fi
