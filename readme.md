# Prometheus Network Shim

This package creates a shim to fill in the prometheus pod network metrics when using a container runtime with virtual interfaces. This should be used when the metrics call returns empty pod details.

```shell
kubectl get --raw /api/v1/nodes/NODE/proxy/metrics/cadvisor | grep 'container_network_transmit_bytes_total'

# HELP container_network_transmit_bytes_total Cumulative count of bytes transmitted
# TYPE container_network_transmit_bytes_total counter
container_network_transmit_bytes_total{container="",id="/",image="",interface="cilium_host",name="",namespace="",pod=""} 4.756927e+06 1681685897133
container_network_transmit_bytes_total{container="",id="/",image="",interface="cilium_net",name="",namespace="",pod=""} 1.5337599e+08 1681685897133
container_network_transmit_bytes_total{container="",id="/",image="",interface="eth0",name="",namespace="",pod=""} 3.30484503673e+11 1681685897133
...
container_network_transmit_bytes_total{container="",id="/",image="",interface="lxc_health",name="",namespace="",pod=""} 125842 1681685897133
container_network_transmit_bytes_total{container="",id="/",image="",interface="lxcbfb8a72ac2b8",name="",namespace="",pod=""} 2.3174557e+07 1681685897133
```

Some code has been copied from the [OpenShift Network Metrics Daemon](https://github.com/openshift/network-metrics-daemon) and the [cri-tools project](https://github.com/kubernetes-sigs/cri-tools).

## Supported run times

Below is a list of supported/tested container runtime/cni combinations. The shim should _technically_ work with any runtime/cni as it uses the kubernetes api and crictl, but there is no guarantee. 

| Runtime | CNI    | Interface(s) |
|---------|--------|--------------|
| cri-o   | Cilium | lxcX         |

> Note: This has only been tested with Kubernetes/k8s self-managed cluster. 
> Other orchestration systems such as k3s, Rancher, OpenShift and Docker Swarm have not been tested, nor have any managed solutions such as ECS or GKE.

## Usage

The shim is designed to run as a daemonset across your cluster. It interacts with the host, so it runs as a privileged container, on the host network, with read-only access to the entire host filesystem.

### Pre-installation

Hosts must have the following tools available:

- crictl
- nsenter
- ethtool

You will also need to remap the original cadvisor `container_network_*` metrics to `network:container_network_*`. A remapping to do this [is supplied](https://github.com/sjdaws/prometheus-network-shim/blob/main/install/kubernetes/prometheus-cadvisor-relabel.yaml) in the install folder.

### Installation

Two manifests are supplied in the install folder.

- [prometheus-network-shim](https://github.com/sjdaws/prometheus-network-shim/blob/main/install/kubernetes/prometheus-network-shim.yaml) will install the shim as a DaemonSet, create a ClusterRole/Binding, and a ServiceMonitor
- [prometheus-rules](https://github.com/sjdaws/prometheus-network-shim/blob/main/install/kubernetes/prometheus-rules.yaml) will join the cadvisor `network:container_network_*` with the shim to create `container_network_*` metrics

These manifests are mapped to the `monitoring` namespace, this can be changed to any namespace you desire.

### Out of cluster

The shim can be run out of cluster, but it will be unable to fetch interface information unless it is running on the node which has the container.

### Post-installation

Once installed, the shim should decorate the network metrics with pod information.

```shell
container_network_transmit_bytes_total{container="mycontainer", endpoint="https-metrics", id="/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod1cdae767_00ab_4a34_ba55_c2a642098751.slice/crio-de374a2de9762915ecf051baccdd29a65e8eacdc00c174a8f9e440dd344bdc10.scope", image="docker.io/alpine:latest", instance="10.0.0.0:10250", interface="lxcbfb8a72ac2b8", job="kubelet", metrics_path="/metrics/cadvisor", name="k8s_mycontainer_mycontainer-94b99dbbf-wsdgr_mynamespace_1cdae767-00ab-4a34-ba55-c2a642098751_0", namespace="mynamespace", node="node1", pod="mycontainer-94b99dbbf-wsdgr", service="kube-prometheus-stack-kubelet"}
```

You can check the shim in prometheus by running the query `pod_interface_shim`.