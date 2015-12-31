#!/bin/bash

# render virtualhost template using values from Etcd and reload Nginx
confd -onetime -backend etcd -node http://etcd:4001
