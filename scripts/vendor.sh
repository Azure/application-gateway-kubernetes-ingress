#!/bin/bash
rm -rf vendor
go mod vendor
chmod -R 777 vendor
