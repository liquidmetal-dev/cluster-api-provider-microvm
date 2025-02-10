# Cluster API provider Microvm

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/liquidmetal-dev/cluster-api-provider-microvm)
[![Go Report Card](https://goreportcard.com/badge/github.com/liquidmetal-dev/cluster-api-provider-microvm)](https://goreportcard.com/report/github.com/liquidmetal-dev/cluster-api-provider-microvm)
[![Slack](https://img.shields.io/badge/join%20slack-%23liquid--metal-brightgreen)](https://weave-community.slack.com/archives/C02KARWGR7S)

------

## What is the Cluster API Provider Microvm

The [Cluster API][cluster_api] brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.

Cluster API Provider Microvm (CAPMVM) is a __Cluster API Infrastructure Provider__ for provisioning Kubernetes clusters where the nodes (control plane & worker) are lightweight virtual machines (called **microvms**). The provider is designed to work with [Flintlock][flintlock] which handles the interaction with the microvm implementation (i.e. [Firecracker][firecracker], [Cloud Hypervisor][cloudhypervisor]).

CAPMVM is [MPL-2.0 licensed](license)

## Features

- Native Kubernetes manifests and API.
- Manages provisioning of microvms via Flintlock.
- Supports specifying custom volume & kernel images.
- Supports specifying the specs of the microvms.

## Getting started

A getting started guide will be available soon.

------

## Compatibility with Flintlock

When using CAPMVM as part of a Liquid Metal system, check the flintlock<->capmvm
[version compatibility](docs/compatibility.md).

------
## Getting Help

If you have any questions about, feedback for or problems with CAPMVM:

- [File an issue](CONTRIBUTING.md#opening-issues).

Your feedback is always welcome!

------
## Contributing

Contributions are welcome. Please read the [CONTRIBUTING.md][contrib] and our [Code Of Conduct][coc].

You can reach out to the maintainers and other contributors using the [#liquid-metal](https://weave-community.slack.com/archives/C02KARWGR7S) slack channel.

Other interesting resources include:

- [The issue tracker][issues]
- [The list of milestones][milestones]
- [Architectural Decision Records (ADR)][adr]

### Our Contributors

Thank you to our contributors:

<p>
<a href="https://github.com/liquidmetal-dev/cluster-api-provider-microvm/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=liquidmetal-dev/cluster-api-provider-microvm" />
</a>
</p>

<!-- References -->
[prow]: https://go.k8s.io/bot-commands
[new_issue]: https://github.com/liquidmetal-dev/cluster-api-provider-microvm/issues/new/choose
[cluster_api]: https://github.com/kubernetes-sigs/cluster-api
[tilt]: https://tilt.dev
[cluster_api_tilt]: https://master.cluster-api.sigs.k8s.io/developer/tilt.html
[cluster-api-supported-v]: https://cluster-api.sigs.k8s.io/reference/versions.html
[flintlock]: https://github.com/liquidmetal-dev/flintlock
[firecracker]: https://firecracker-microvm.github.io/
[cloudhypervisor]: https://www.cloudhypervisor.org/
[contrib]: ./CONTRIBUTING.md
[coc]: ./CODE_OF_CONDUCT.md
[milestones]: https://github.com/liquidmetal-dev/cluster-api-provider-microvm/milestones
[adr]: ./docs/adr
[license]: ./LICENSE
[issues]: https://github.com/liquidmetal-dev/cluster-api-provider-microvm/issues
