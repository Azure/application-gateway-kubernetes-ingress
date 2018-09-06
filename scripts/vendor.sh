#!/bin/bash
rm -rf vendor
glide install -v
chmod -R 777 vendor
