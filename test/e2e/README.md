# End-to-end testing

The e2e tests are designed to be run in a number of ways.
If you are developing CAPMVM, continue reading this doc.
If you are testing the whole of Liquid Metal, check out the [Liquid Metal Acceptance Tests][lmats]
docs.

## Start flintlock

You will need a running flintlock server. This can be done locally, if you are
working on Linux. Mac or windows users have the option to run flintlock on
an [Equinix][equinix] host, but if you do not have an account there you will not
be able to run these tests.

```bash
git clone https://github.com/liquidmetal-dev/flintlock
cd flintlock
sudo ./hack/scripts/provision.sh --grpc-address 0.0.0.0:9090 --dev --insecure
```

This will clone flintlock and bootstrap your machine to run a server. You can
read the [flintlock docs][fl-docs] if you would like to set this up manually
and see each individual step. **Make sure you start flintlock with `--grpc-address`
set to `0.0.0.0:9090` otherwise CAPMVM will not be able to reach it from within
the `kind` network.**

If you ran the above command, flintlock will be running as a `systemd` service.

## DHCP

When microvms are created they will request an IP from a DHCP server. Your router
should have one, but you may need to check the settings or start a new server
for the purpose of the tests.

TODO: explain this

## Required tools

Ensure you have the following installed:
- [kind](https://kind.sigs.k8s.io/)
- [docker](https://docs.docker.com/engine/install/ubuntu/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [clusterctl](https://cluster-api.sigs.k8s.io/user/quick-start.html#install-clusterctl)

## Run the tests

In your CAPMVM repo, save your flintlock server address to a variable and run
the tests:

```bash
# your flintlock server should be bound to your machine's private address
# you can get that with hostname
FL=$(hostname -I | awk '{print $1}')
make e2e E2E_ARGS="-e2e.flintlock-hosts $FL:9090" # don't forget the port!
```

_Note: the tests will default to using `192.168.1.15` for the workload cluster's
load balancer IP. You will need to verify that this is within your network and
unused, or configure the tests to use another by setting the `-e2e.capmvm.vip-address`
flag. See "Test options" below for details._

The tests will take ~5 mins to run.

They will:
- Create a new kind cluster
- Init the cluster with CAPI providers
- Generate a CAPMVM workload cluster
- Apply the cluster
- Wait for the control plane and the worker nodes to start
- Verify that all given flintlock hosts were used (if you can start a second
	flintlock server on a different port, you can pass both to the tests with
	`make e2e E2E_ARGS="-e2e.flintlock-hosts 1.2.3.4:9090,4.5.6.7:9091"`)
- Deploy nginx to the workload cluster
- Ensure that nginx starts successfully
- Delete the workload cluster
- Delete the kind cluster

To speed up your testing cycle, you can pass the `-e2e.existing-cluster` flag.
See "Test options" below for details.

## Test options

The following flags are available:

```
Usage of /home/claudia/workspace/cluster-api-provider-microvm/test/e2e/e2e.test:
  -e2e.artefact-dir string
        Location to store test yamls, logs, etc. (default "/home/claudia/workspace/cluster-api-provider-microvm/test/e2e/_artefacts")
  -e2e.capmvm.kubernetes-version string
        Version of k8s to run in the workload cluster(s) (default "1.21.8")
  -e2e.capmvm.vip-address string
        Address for the kubevip load balancer. (default "192.168.1.25")
  -e2e.config string
        Path to e2e config for this suite. (default "config/e2e_conf.yaml")
  -e2e.existing-cluster
        If true, uses the current context for the management cluster and will not create a new one.
  -e2e.flintlock-hosts value
        Comma separated list of addresses to flintlock servers. eg '1.2.3.4:9090,5.6.7.8:9090'
  -e2e.skip-cleanup
        Do not delete test-created workload clusters or the management kind cluster.
```

These can be passed to the `make` command:

```bash
make e2e E2E_ARGS="-e2e.skip-cleanup"
```

To use the `e2e.existing-cluster` boolean flag, you will need to ensure that the
cluster is set as the current context.

_Note: `-e2e.flintlock-hosts` and `-e2e.artefact-dir` are already passed to the
tests as part of the `make` command._

[lmats]: https://github.com/liquidmetal-dev/liquid-metal-acceptance-tests
[equinix]: https://metal.equinix.com/
