#!/bin/bash
aws s3api list-objects --bucket "ndex.cytoscape.io" --output json --query "[sum(Contents[].Size), length(Contents[])]"
