# daemonset-job-terminator

[![CircleCI](https://circleci.com/gh/darkowlzz/daemonset-job-terminator.svg?style=svg)](https://circleci.com/gh/darkowlzz/daemonset-job-terminator)

Sidecar part of the daemonset-job k8s operator. This must be deployed along with
the daemonset-job operator. It monitors the pods created by daemonset-job and
terminates the parent Job, which results cleanup of all the resources by garbage
collection of the DaemonSet and its Pods.

## Development

Build the container image:
```
$ docker build . -t darkowlzz/daemonset-job-terminator:latest
```

Deploy the sidecar independent of any operator:
```
$ kubectl apply -f deploy/pod.yaml
```
