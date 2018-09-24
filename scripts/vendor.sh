#!/bin/bash
rm -rf vendor
glide update -v
chmod -R 777 vendor
