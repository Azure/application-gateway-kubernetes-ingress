#!/usr/bin/env python3

"""
This script updates index.yaml so that the appVersion key of the latest entry has the value of the latest git tag.
"""

import subprocess
import yaml

FILE = 'index.yaml'

git_tag = subprocess.check_output('git describe --abbrev=0 --tags', shell=True).decode('utf-8').strip('\n')

print(f'Updating appVersion to {git_tag}')

with open(FILE, 'r') as f:
    index_yaml = yaml.load(f)

for record in index_yaml['entries']['ingress-azure']:
    if record['version'] == git_tag:
        record['appVersion'] = git_tag
        break

with open(FILE, 'w') as f:
    yaml.dump(index_yaml, f)
