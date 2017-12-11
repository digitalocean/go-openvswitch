#!/bin/bash

# Verify that all files are correctly gofmt'd, with the exception of
# generated code.
EXIT=0
GOFMT=$(go fmt ./... | grep -v "ovsnl/internal/ovsh")

if [[ ! -z $GOFMT ]]; then
	echo -e "Files that are not gofmt'd:\n"
	echo "$GOFMT"
	EXIT=1
fi

exit $EXIT
