#!/bin/bash
set -x

helm template ./app/ > generated.yaml 