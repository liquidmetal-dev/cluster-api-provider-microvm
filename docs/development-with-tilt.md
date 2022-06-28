# Developing with Tilt

This guide show how you can use **Tilt** for interactive development/debugging.

## Install Pre-requisites

* [Go](https://go.dev/doc/install)
* [Docker](https://www.docker.com/get-started)
    * Be aware of the licencing changes for Docker
* [kind](https://github.com/kubernetes-sigs/kind)
* [kustomize](https://github.com/kubernetes-sigs/kustomize)
* [envsubst](https://github.com/a8m/envsubst)
* [tilt](https://docs.tilt.dev/install.html)

## Get the source

> Historically there have been issues with some of the code generation tools used when running them on projects outside the **GOPATH**. Therefore the instructions below use the GOPATH.

1. Fork the CAPMVM repo.
2. Open a terminal and get the source code for your fork:

    ```bash
    cd $(go env GOPATH)
    mkdir -p src/github.com/weaveworks
    cd src/github.com/weaveworks
    git clone git@github.com:<GITHUBUSERNAME>/cluster-api-provider-microvm.git
    cd cluster-api-provider-microvm
    git remote add upstream git@github.com:weaveworks-liquidmetal/cluster-api-provider-microvm.git
    git fetch upstream
    ```

3. Open a second terminal window and get the source code for CAPI:

    ```bash
    cd $(go env GOPATH)
    mkdir -p src/sigs.k8s.io
    cd src/sigs.k8s.io
    git clone git@github.com:kubernetes-sigs/cluster-api.git
    ```

## Configure Tilt

In your cluster-api folder create a file called **tilt-settings.json**:

```json
{
    "default_registry": "gcr.io/yourusername",
    "provider_repos": ["PATH/TO/YOUR/CAPMVM"],
    "enable_providers": ["microvm", "kubeadm-bootstrap", "kubeadm-control-plane"],
    "kustomize_substitutions": {
        "EXP_MACHINE_POOL": "true",
        "EXP_CLUSTER_RESOURCE_SET": "true"
    },
    "extra_args": {
        "microvm": ["--v=4"],
        "kubeadm-control-plane": ["--v=4"],
        "kubeadm-bootstrap": ["--v=4"],
        "core": ["--v=4"]
    },
    "debug": {
        "microvm": {
            "continue": true,
            "port": 30000
        }
    }
}
```

For a full list of the options see the [docs](https://cluster-api.sigs.k8s.io/developer/tilt.html).

## Create a cluster & start tilt

We will run tilt in a kind based cluster.

1. Open a terminal
2. Change directory to the **cluster-api** folder
3. Run the following:

    ```bash
    # if you have any GITHUB_TOKEN or cred set in the environment, unset that first

    export CAPI_KIND_CLUSTER_NAME=capmvm-test
    kind create cluster --name $CAPI_KIND_CLUSTER_NAME
    tilt up
    ```

When tilt is started you can press the **spacebar** to open up a browser based UI.

## Start flintlock

Ensure that you have an instance of flintlock (and containerd) [configured and running](https://github.com/weaveworks-liquidmetal/flintlock/blob/main/docs/quick-start.md).
Be sure to start flintlock with `--grpc-endpoint=0.0.0.0:9090` or the CAPMVM controller
will not be able to connect to the server from within the Kind cluster.

## Create cluster definition

Create the declaration for your cluster. We will use the template in the repo.

1. Open a terminal and go to your **cluster-api-provider-microvm** folder
2. Get the IP address of your running flintlock server
3. Create a cluster declaration from the template

    ```bash
    export KUBERNETES_VERSION=v1.20.0
    export CLUSTER_NAME=mvm-test
    export CONTROL_PLANE_MACHINE_COUNT=1
    export WORKER_MACHINE_COUNT=1
    export CONTROL_PLANE_VIP=192.168.8.15
    export MVM_ROOT_IMAGE=docker.io/richardcase/ubuntu-bionic-test:cloudimage_v0.0.1
    export MVM_KERNEL_IMAGE=docker.io/richardcase/ubuntu-bionic-kernel:0.0.11
    # NOTE: change 192.168.8.2 to be the IP address from step 2
    export HOST_ENDPOINT=192.168.8.2:9090

    cat templates/cluster-template.yaml | envsubst > cluster.yaml
    ```

4. Edit **cluster.yaml** to make any changes or comment out sections.

5. Do a dry run for the manifests:

    ```bash
    kubectl apply -f cluster.yaml --dry-run=server
    ```

6. If you are happy apply and watch the logs in the tilt ui.

    > Tilt uses hot reloading so you can make code changes to CAPMVM and it will automatically rebuild/deploy capmvm.

7. To delete the cluster **do not** use `kubectl delete -f cluster.yaml`.
    Run `kubectl delete clusters.cluster.x-k8s.io mvm-test`.

## Debug

If you want to attach a debugger to capmvm this section in **tilt-settings.json** configures tilt to start the provider using **delve** and listen on port 30000.

> If you don't intent on debugging then you can remove this section.

You can connect to the running instance of delve. If you are using vscode then you can use a launch configuration like this:

```json
    {
        "name": "Connect to tilt",
        "type": "go",
        "request": "attach",
        "mode": "remote",
        "remotePath": "",
        "port": 30000,
        "host": "127.0.0.1",
        "showLog": true,
        "trace": "log",
        "logOutput": "rpc"
    }
```

Or you can connect using delve on the command line (_so the legend goes, I have not been able to get it to work yet_):

```bash
dlv connect 127.0.0.1:30000
```
