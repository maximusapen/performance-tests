# Tools
A location for random tools

## Network traffic disruptors
* deny-compose-egress.yml - Calico policy to stop traffic to compose. Net and port need to be changed for different carriers.
* deny-compose-ingress.yml - Calico policy to stop traffic from compose. Net and port need to be changed for different carriers.
* disable-nfs.yml - Calico policy to stop nfs traffic.
* disableNFS.sh - Updates iptables to force nfs packets to be dropped. Usage: disableNFS.sh
* dropPackets.sh - Updates iptables to force packets to be dropped. Usage: dropPackets.sh port[,port[,port]] [ip]

## Other
* nfsiostat_parser.py
* delete-stranded-IAM-IDs.sh - Only best effort is used to delete IDs when cluster is deleted. Cleanup is needed to keep total number of IDs below the 2000 limit.
