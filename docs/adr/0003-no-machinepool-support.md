# Support Machine (and MachineTemplate) but not MachinePool

* Status: accepted
* Date: 2021-11-1
* Authors: @richardcase
* Deciders: @jmickey @richardcase @Callisto13 @yitsushi

## Context

Cluster API[^1][^2] supports a number of resources kinds that infrastructure providers can implement to create machines for use as Kubernetes nodes:

* **Machine** - is a declaration for an individual machine. There is always an infrastructure provider counterpart (e.g. AWSMachine, AzureMachine) that knows how to provision the infrastructure for a machine based on a spec. **Machines** are immutable and so the actions required will only be create/delete.
* **MachineTemplate** - is a declaration for a template of a machine. It doesn't relate to a specific machines but the spec it contains will be used to create instances of **Machines** when using the Kubeadm control plane.
* **MachinePools** - are an experimental feature in CAPI and provide a way to declare a dynamic pool of machines where the number of instances can scale up and down. An infrastructure provider provides this via a specific solution to the infrastructure. So CAPA uses Auto Scale Groups (ASG) and CAPZ uses Virtual Machine Scale Sets.

## Decision

Cluster API Provider Microvm (CAPMVM) will not support **Machine Pools** initially as [flintlock](https://github.com/liquidmetal-dev/flintlock) has no way concept of auto-scaling or machine pools.

## Consequences

We will need to implement the following infrastructure kinds:

* MicrovmCluster (infrastructure counterpart to Cluster)
* MicrovmMachine (infrastructure counterpart to Machine)
* MicrovmMachineTemplate (infrastructure machine template used by Kubeadm Bootstrap Provider)

We will need to revisit this decision when flintlock supports auto-scaling / pools.

[^1]: https://cluster-api.sigs.k8s.io/introduction.html
[^2]: https://github.com/kubernetes-sigs/cluster-api
