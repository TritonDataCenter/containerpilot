FROM quay.io/coreos/etcd:v2.0.8

EXPOSE 2379 2380 4001

CMD [\
    "-name","etcd0",\
    "-advertise-client-urls","http://etcd:2379,http://etcd:4001",\
    "-listen-client-urls","http://0.0.0.0:2379,http://0.0.0.0:4001",\
    "-initial-advertise-peer-urls","http://etcd:2380",\
    "-listen-peer-urls","http://0.0.0.0:2380",\
    "-initial-cluster-token","etcd-cluster-1",\
    "-initial-cluster","etcd0=http://etcd:2380",\
    "-initial-cluster-state","new"]
