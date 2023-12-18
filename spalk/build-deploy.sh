#!/bin/bash
cd $GOPATH/src/github.com/SpalkLtd/grafana-oss
docker build -t grafana_local . && docker tag grafana_local:latest 496668274218.dkr.ecr.us-east-1.amazonaws.com/prod_grafana:current  && docker push 496668274218.dkr.ecr.us-east-1.amazonaws.com/prod_grafana:current

