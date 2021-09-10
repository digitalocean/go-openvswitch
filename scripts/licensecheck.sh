#!/bin/bash 

# Verify that the correct license block is present in all Go source
# files.
LICENSE=./scripts/license.txt
NB_LINES=$(cat $LICENSE| wc -l)
EXPECTED=$(head -$NB_LINES $LICENSE | tail +2)

# Update the string if there is a new year to consider
COPYRIGHT_YEARS="2017 2021"

# Scan each Go source file for license.
EXIT=0
GOFILES=$(find . -name "*.go")

for FILE in $GOFILES; do
	# Start validating the Copyright line of for each header.
	# Years can change from a source file to another.
	read -r -a COPYRIGHT_TOKENS <<< $(head -n 1 $FILE |awk '{print $2 " " $3 " " $4}')
	if [ "${COPYRIGHT_TOKENS[0]}" != "Copyright" ]; then
		echo "Bad Copyright token ${COPYRIGHT_TOKENS[0]} in $FILE"	
		EXIT=1
	elif [ $(echo $COPYRIGHT_YEARS | grep -c ${COPYRIGHT_TOKENS[1]}) != "1" ]; then
		echo "Bad Year token ${COPYRIGHT_TOKENS[1]} in $FILE"	
		EXIT=1
	elif [ "${COPYRIGHT_TOKENS[2]}" != "DigitalOcean." ]; then
		echo "Bad DigitalOcean token ${COPYRIGHT_TOKENS[2]} in $FILE"	
		EXIT=1
	fi

	BLOCK=$(head -n 14 $FILE|tail +2)

	if [ "$BLOCK" != "$EXPECTED" ]; then
		echo "file missing license: $FILE"
		EXIT=1
	fi
done

exit $EXIT
