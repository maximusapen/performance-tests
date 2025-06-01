#!/bin/bash 
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
#set -e
#    Copyright (C) 2012 Dustin Kirkland <kirkland@ubuntu.com>
#
#    Authors: Dustin Kirkland <kirkland@ubuntu.com>
#             Kees Cook <kees@ubuntu.com>
#
#    This program is free software: you can redistribute it and/or modify
#    it under the terms of the GNU General Public License as published by
#    the Free Software Foundation, version 3 of the License.
#
#    This program is distributed in the hope that it will be useful,
#    but WITHOUT ANY WARRANTY; without even the implied warranty of
#    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#    GNU General Public License for more details.
#
#    You should have received a copy of the GNU General Public License
#    along with this program.  If not, see <http://www.gnu.org/licenses/>.


# Ensure we're running as root
if [ "$(id -u)" != 0 ]; then
    echo "ERROR: This script must run as root.  Hint..." 1>&2
    echo "  sudo $0 $@" 1>&2
    exit 1
fi
# NOTE: This script will ALWAYS keep the currently running kernel
# NOTE: Default is to keep 2 more, user overrides with --keep N
KEEP=2
# NOTE: Any unrecognized option will be passed straight through to apt
APT_OPTS=
while [ ! -z "$1" ]; do
    case "$1" in
        --keep)
            # User specified the number of kernels to keep
            KEEP="$2"
            shift 2
        ;;
        *)
            APT_OPTS="$APT_OPTS $1"
            shift 1
        ;;
    esac
done
# Print all kernal packages before purge
echo "All kernal packages in /boot - Will keep 2 with current included"
ls -tr /boot/vmlinuz-*
# Build our list of kernel packages to purge
CANDIDATES=$(ls -tr /boot/vmlinuz-* | head -n -${KEEP} | grep -v "$(uname -r)$" | cut -d- -f2- | awk '{print "linux-image-" $0 " linux-headers-" $0}' )
for c in $CANDIDATES; do
    dpkg-query -s "$c" >/dev/null 2>&1 && PURGE="$PURGE $c"
done
if [ -n "$PURGE" ]; then
   echo "Purging $PURGE"
   apt $APT_OPTS -y remove --purge $PURGE    
fi
# end of kernel cleanup

