# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

BEGIN {
	TIME="time"
	ITEM="item"
	OP="op"
}
/configmap|namespace|secret|service/ {
	print $1 "," $1 ",0m0s," $2 ":" $3
} 
!/namespace/ {
	if  ( $4 == "created" ) {
		TIME=$1
		ITEM=$2
		OP=$3
	} else {
		command="date -d \"" TIME "\" +%s"
		command | getline StartEpoch
		close(command)
		command="date -d \"" $1 "\" +%s"
		command | getline EndEpoch
		close(command)
		DELTA=EndEpoch-StartEpoch
		SECONDS=DELTA % 60
		MINUTES=(DELTA-SECONDS) / 60
		print TIME "," $1 "," MINUTES "m" SECONDS  "s," $2 ":" $3
	}
}
