# Minimizing Downtime During Deployments

## Purpose
This document outlines a Kubernetes and Ingress controller configuration, which when incorporated with proper Kubernetes rolling updates deployment could achieve a near-zero-downtime deployments.

## Overview
It is not uncommon for Kubernetes operators to observe Application Gateway 502 errors while performing a Kubernets rolling update on an AKS cluster fronted by Application Gateway and AGIC. This document offers a method to alleviate this problem. Since the method described in this document relies on correctly aligning the timing of deployment events it is not possible to guarantee 100% elimination of the probability of running into a 502 error. Even with this method there will be a non-zero chance for a period of time where Application Gateway backends could lag behind the most recent updates applied by a rolling update to the Kubernetes pods.

## Understanding 502 Errors
At a high level there are 3 scenarious in which one could observe 502 errors on an AKS cluster fronted with App Gateway and AGIC. In all of these the root cause is the delay one could observe in applying a IP address changes to the Application Gateway's backend pools.

  - Scaling down a Kubernetes cluster:
    - Kubernetes is instructed to lower the number of pod replicas (perhaps manually, or via Horizontal Pod Autoscaler, or some other mechanism)
    - Pods are put in Terminating state, while simultaneously removed from the list of Endpoints.
    - AGIC observes the fact that Pods + Endpoints changed and begins a config update on App Gateway
     It takes somewhere between a second and a few minutes for a pod, or a list of the pods to be removed from App Gateway's backend -- meanwhile App Gateway still attempts to deliver traffic to terminated pods
    - Result is occasional 502 errors

  - Rolling Updates:
      - Customer updates the version of the software (perhaps using `kubectl set image`)
      - Kubernetes upgrades a percentage of the pods at a time. The size of the bucket is defined in [the strategy section of the Deployment spec](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#updating-a-deployment)
         - Kubernetes adds a new pod with a new image - pod goes through the states from `ContainerCreating` to `Running`
         - When the new pod is in Running state - Kubernetes terminates the old pod
       - The process described above is repeated until all pods are upgraded

  - Kubernetes terminates resource-starved pods (CPU, RAM etc)


## Solution
The solution below lowers the probability of running into a scenario where App Gateway's backend pool points to terminated pods, resulting in 502 error. The solution below does not completely remove this chance.

Required configuration changes prior to [performing a rolling update](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/):


1. Change the Pod and/or Deployment specs by adding [preStop container life-cycle hooks](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks), with a delay (sleep) of at least 90 seconds.
Example:
```yaml
kind: Deployment
metadata:
  name: x
  labels:
    app: y
spec:
  ...
  template:
    ...
    spec:
      containers:
      - name: ctr
        ...
        lifecycle:
          preStop:
            exec:
              command: ["sleep","90"]
```


> Note: The "sleep" command assumes the container is based on Linux. For Windows containers the equivalent command is `["powershell.exe","-c","sleep","90"]`.

The addition of the `preStop` container life cycle hook will:
  - delay Kubernetes sending `SIGTERM` to the container by 90 seconds, but put the pod immediately in `Terminating` state
  - simultaneously this will also immediately remove the pod from the Kubernetes Endpoints list
  - this will cause AGIC to remove the pod from App Gateway's backend pool
  - pod will continue to run for the next 90 seconds - giving App Gateway 90 seconds to execute "remove from backend pools" command



2. Add [connection draining annotation](https://docs.microsoft.com/bs-latn-ba/azure/application-gateway/ingress-controller-annotations#connection-draining) to the Ingress read by AGIC to allow for in-flight connections to complete. Example:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
 name: websocket-ingress
 annotations:
   kubernetes.io/ingress.class: azure/application-gateway
   appgw.ingress.kubernetes.io/connection-draining: "true"
   appgw.ingress.kubernetes.io/connection-draining-timeout: "30"
```

What this achieves - when a pod is pulled from an App Gateway backend it will disappear from the UI, but existing in-flight connections will not be immediately terminated -- they will be given 30 seconds to complete.

We believe that the addition of the `preStop` hook and the connection draining annotation will drastically remove the probability for App Gateway to attempt to connect to a terminated pod.


3. Add [terminationGracePeriodSeconds](https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#hook-handler-execution) to the Pod resource YAML. This must be set to a value that is greater than the `preStop` hook wait time.

```yaml
kind: Deployment
metadata:
  name: x
  labels:
    app: y
spec:
  ...
  template:
    ...
    spec:
      containers:
      - name: ctr
        ...
      terminationGracePeriodSeconds: 101
```

4. Decrease interval between App Gateway health probes to backend pools. The goal is to increase number of probes per unit of time. This will ensure that a terminated pod, which has not yet been removed from App Gateway's backend pool, will be marked as unhealthy sooner, thus removing the probability of a request landing on a terminated pod and resulting in a 502 error.

For example the following [Kubernetes Deployment liveness probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/) will result in the respective pods being marked as unhealthy after 15 seconds and 3 failed probes. This config will be directly applied to Application Gateway (by AGIC), as well as Kubernetes.

```yaml
...
        livenessProbe:
          httpGet:
            path: /
            port: 80
          periodSeconds: 4
          timeoutSeconds: 5
          failureThreshold: 3
```

## Summary
To achieve a near-zero-downtime deployments, we need to add a:
  - `preStop` hook waiting for 90 seconds
  - termination grace period of at least 90 seconds
  - connection draining timeout of about 30 seconds
  - aggressive health probes

> Note: All proposed parameter values above should be adjusted for the specifics of the system being deployed.


Long term solutions to zero-downtime updates:
  1. Faster backend pool updates: The AGIC team is already working on the next iteration of the Ingress Controller, which will shorten the time to update App Gateway drastically. Faster backend pool updates will lower the probability to run into 502s.
  2. Rolling updates with App Gateway feedback: AGIc team is looking into a deeper integration between AGIC and the Kubernetes' rolling updates feature.
