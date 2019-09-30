#!/bin/bash

grep 'errors.New' $(find . -name '*.go' -not -path './git/*' -not -path './vendor/*' | grep -v '_test.go' | grep -v 'crd_client') | grep -oE '\([A-Z]{4}[0-9]{3}\)' | sort | uniq -c |  awk '{if ($1 > 1) print $0}'
