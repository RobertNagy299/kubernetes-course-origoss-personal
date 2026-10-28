# Fault hunt report

Fill in one section per fault, in the order you found them. Paste real output, not a summary.

## Fault 1

- **Symptom** (what you saw first): Every single pod is `ready 0/1` and is in a pending state. EG: None of our pods work, and our app is down.
- **Command** (that showed the cause): `kubectl get pods`
- **Evidence** (the exact output line):

| NAME | READY | STATUS | RESTARTS | AGE |
| :--- | :--- | :--- | :--- | :--- |
| hello-7f6755ddc7-tfs6d | 0/1 | Pending | 0 | 3h16m |
| hello-7f6755ddc7-z8qcb | 0/1 | Pending | 0 | 3h16m |


- **Cause**: (Too early to tell, I gotta dig deeper. See what I do under the next section - Fault 2)
- **Fix** (file and field you changed): (too early to tell, gotta dig deeper --> Fault 2)

## Fault 2

- **Symptom** (what you saw first): Upon seeing non of our pods ever came alive, I checked the events. I saw that the pods have insufficient cpu
- **Command** (that showed the cause): `kubectly events`
- **Evidence** (the exact output line):
`96s (x5 over 21m)       Warning   FailedScheduling                    Pod/hello-7f6755ddc7-z8qcb      0/1 nodes are available: 1 Insufficient cpu. preemption: 0/1 nodes are available: 1 Preemption is not helpful`
- **Cause**: Upon seeing the deployment.yaml file, I saw that under resources:requests:cpu: we see "64", but there is no unit of measurement.
- **Fix** (file and field you changed): line 27 in deployment.yaml. I added a unit: millicores. I changed `cpu: "64"` to `cpu: "64m"`

## Fault 3

- **Symptom** (what you saw first): Upon seeing the events, I opened the deployment.yaml file to update the CPU requests. Upon opening the file, I noticted that under the spec:containers:image section, the wrong image is specified (hello:0.1.10 instead of hello:0.1.1)
- **Command** (that showed the cause): `code .` (I opened the whole folder in VS code). But if we only have terminal access, we can use `cat deployment.yaml`.
 - **Evidence** (the exact output line):
 ```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  labels:
    app: hello
spec:
  replicas: 2
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
        - name: hello
          image: hello:0.1.10 <-- this right here, officer

 ```
- **Cause**: possible typo
- **Fix** (file and field you changed): `deployment.yaml` line 19 (spec:template:spec:containers:image) -> changed from `image: hello:0.1.10` to `image:hello:0.1.1`.

## Fault 4

- **Symptom** (what you saw first): After updating the image and the cpu request in the yaml file, I applied the file and checked the rollout status. I saw that the rollout status looked off: `Waiting for deployment "hello" rollout to finish: 1 out of 2 new replicas have been updated...`
I immediatelly knew this means something failed, so I checked the pods and I saw a new pod next to the old ones. The new pod's status was ErrCrashLoopBackOff so I took a look at the events.
- **Command** (that showed the cause):

It was a chain of commands:
```
kubectl apply -f deployment.yaml
kubectl get deployments
kubectl rollout status deployment/hello
kubectl get pods
kubectl events
```
- **Evidence** (the exact output line):
```
35s (x5 over 2m2s)      Normal    Created                             Pod/hello-5bd48dd7cb-gkpxn      Container created
35s (x5 over 2m2s)      Normal    Pulled                              Pod/hello-5bd48dd7cb-gkpxn      Container image "hello:0.1.1" already present on machine and can be accessed by the pod
34s (x5 over 2m1s)      Warning   Failed                              Pod/hello-5bd48dd7cb-gkpxn      Error: failed to create containerd task: failed to create shim task: OCI runtime create failed: runc create failed: unable to start container process: container init was OOM-killed (memory limit too low?)
32s (x16 over 2m)       Warning   BackOff                             Pod/hello-5bd48dd7cb-gkpxn      Back-off restarting failed container hello in pod hello-5bd48dd7cb-gkpxn_default(41efbd2a-a0df-402e-96eb-f54c790ad807)

```
- **Cause**: OOM-killed - memory limit too low ?
- **Fix** (file and field you changed): First, I will try increasing the memory limit from 2Mi to 128Mi in the deployment.yaml file, and then apply the file. So I changed line 29 from `memory: 2Mi` to `memory: 128Mi`

## Fault 5

- **Symptom** (what you saw first): After increasing the memory limit, and redeploying the app, I saw the same crash error as before. So I ran pretty much the same chain of commands
- **Command** (that showed the cause): `kubectl events` but only after using `kubectl rollout status deployment/hello` and `kubectl get pods`
- **Evidence** (the exact output line):

```
13s (x3 over 26s)        Normal    Created                             Pod/hello-7bcd89485f-4jcmq      Container created
13s (x3 over 26s)        Normal    Pulled                              Pod/hello-7bcd89485f-4jcmq      Container image "hello:0.1.1" already present on machine and can be accessed by the pod
13s (x3 over 25s)        Normal    Started                             Pod/hello-7bcd89485f-4jcmq      Container started
12s (x3 over 25s)        Warning   Unhealthy                           Pod/hello-7bcd89485f-4jcmq      Startup probe failed: Get "http://10.244.0.6:8080/healthz": dial tcp 10.244.0.6:8080: connect: connection refused
7s (x8 over 23s)         Warning   BackOff                             Pod/hello-7bcd89485f-4jcmq      Back-off restarting failed container hello in pod hello-7bcd89485f-4jcmq_default(f5aaf20b-7bfd-41dd-b0ab-a363b631bcd4)

```
- **Cause**: The error message says that the startup probe failed, because the connection failed.
- **Fix** (file and field you changed): My first thought was to look into the deployment.yaml file and check all the ports. I saw that the env:value: was "abc" which is an invalid port. So I changed line 22 in deployment.yaml to `value: 8080`. After trying to apply the file, I got this error: 

```

kubectl apply -f deployment.yaml 
The request is invalid: patch: Invalid value: "map[metadata:map[annotations:map[kubectl.kubernetes.io/last-applied-configuration:{\"apiVersion\":\"apps/v1\",\"kind\":\"Deployment\",\"metadata\":{\"annotations\":{},\"labels\":{\"app\":\"hello\"},\"name\":\"hello\",\"namespace\":\"default\"},\"spec\":{\"replicas\":2,\"selector\":{\"matchLabels\":{\"app\":\"hello\"}},\"template\":{\"metadata\":{\"labels\":{\"app\":\"hello\"}},\"spec\":{\"containers\":[{\"env\":[{\"name\":\"PORT\",\"value\":8080}],\"image\":\"hello:0.1.1\",\"livenessProbe\":{\"httpGet\":{\"path\":\"/healthz\",\"port\":8008},\"periodSeconds\":10},\"name\":\"hello\",\"ports\":[{\"containerPort\":8080}],\"readinessProbe\":{\"httpGet\":{\"path\":\"/healthz\",\"port\":8081},\"periodSeconds\":5},\"resources\":{\"limits\":{\"memory\":\"128Mi\"},\"requests\":{\"cpu\":\"64m\"}},\"startupProbe\":{\"failureThreshold\":10,\"httpGet\":{\"path\":\"/healthz\",\"port\":8080},\"periodSeconds\":1}}]}}}}\n]] spec:map[template:map[spec:map[]]]]": cannot convert int64 to string


```
so I went back and changed that line to `value: "8080"` and then the apply command worked.

## Fault 6

- **Symptom** (what you saw first): After changing the env value to "8080", I applied the file and checked the rollout status. It got stuck again, so I ran `get pods` and then I saw a new pod which is now finally Running, but it is still not ready. So I checked the events again.
- **Command** (that showed the cause): chain of commands, pretty much the same as before. Apply, rollout status, get pods, events.
- **Evidence** (the exact output line):

```
6s (x2 over 36s)        Normal    Created                             Pod/hello-d5594f979-jzbmn       Container created
6s (x2 over 37s)        Normal    Pulled                              Pod/hello-d5594f979-jzbmn       Container image "hello:0.1.1" already present on machine and can be accessed by the pod
5s (x2 over 36s)        Normal    Started                             Pod/hello-d5594f979-jzbmn       Container started
2s (x11 over 35s)       Warning   Unhealthy                           Pod/hello-d5594f979-jzbmn       Readiness probe failed: Get "http://10.244.0.7:8081/healthz": dial tcp 10.244.0.7:8081: connect: connection refused
```

- **Cause**: The error states that the readiness probe failed. I can immediately see that it's probably due to it trying on a wrong port.
- **Fix** (file and field you changed): I will change `line 39` in `deployment.yaml`. I will change `port: 8081` to `port: 8080` in order to match the port specified in the environment variable, which indicates the port our web app is using.

## Fault 7

- **Symptom** (what you saw first): Upon fixing the port number for the readinessProbe, I applied the file, and checked the rollout. Things seemed fine:

```
kubectl apply -f deployment.yaml 
deployment.apps/hello configured
robi@debian:~/programming/kubernetes-origoss/kubernetes-course-origoss-personal/fault-hunt$ kubectl rollout status deployment/hello
Waiting for deployment "hello" rollout to finish: 1 out of 2 new replicas have been updated...
Waiting for deployment "hello" rollout to finish: 1 out of 2 new replicas have been updated...
Waiting for deployment "hello" rollout to finish: 1 out of 2 new replicas have been updated...
Waiting for deployment "hello" rollout to finish: 1 old replicas are pending termination...
Waiting for deployment "hello" rollout to finish: 1 old replicas are pending termination...
Waiting for deployment "hello" rollout to finish: 1 old replicas are pending termination...
Waiting for deployment "hello" rollout to finish: 1 old replicas are pending termination...
deployment "hello" successfully rolled out

```
The rollout was a success, and upon checking with kubectl get pods, I see this:

```
kubectl get pods
NAME                     READY   STATUS    RESTARTS   AGE
hello-59fc8dcdfd-4zl7t   1/1     Running   0          21s
hello-59fc8dcdfd-rmx6z   1/1     Running   0          18s

```
I see 2 healthy pods. The old ones were deleted. Time to connect to them.
However, after running curl, I see an error.
- **Command** (that showed the cause): `curl -i localhost:8080/healthz`
- **Evidence** (the exact output line): `curl: (7) Failed to connect to localhost port 8080 after 0 ms: Couldn't connect to server`
- **Cause**: There could be multiple causes here. First of all, I see the livenessProbe is also configured with the wrong port inside deployment.yaml. However, this is not the primary cause here. The issue will probably lie inside the server file, but first we need to fix the liveness probe to make sure kubernetes automatically catches certain issues with its liveness check. The liveness probe issue was also confirmed by kubectl events:

```
52s (x19 over 5m52s)    Warning   Unhealthy                           Pod/hello-59fc8dcdfd-rmx6z      Liveness probe failed: Get "http://10.244.0.9:8008/healthz": dial tcp 10.244.0.9:8008: connect: connection refused


```
- **Fix** (file and field you changed): First, I update the livenessprobe port inside `deployment.yaml` at line `44`. From `port: 8008` to `port: 8080`. This should not fix the connectivity issue, but it might give us some more clues and help narrow things down.
Update: As expected, fixing the livenessprobe did not fix our connection issue. At least it makes the kubectl events nice and clean, no probe errors anymore:

```
22s                     Normal    Pulled                              Pod/hello-7788754f4b-x6747      Container image "hello:0.1.1" already present on machine and can be accessed by the pod
22s                     Normal    Created                             Pod/hello-7788754f4b-x6747      Container created
21s                     Normal    Started                             Pod/hello-7788754f4b-x6747      Container started
21s                     Normal    SuccessfulCreate                    ReplicaSet/hello-7788754f4b     Created pod: hello-7788754f4b-mrs5f
21s                     Normal    SuccessfulDelete                    ReplicaSet/hello-59fc8dcdfd     Deleted pod: hello-59fc8dcdfd-4zl7t
20s                     Normal    Scheduled                           Pod/hello-7788754f4b-mrs5f      Successfully assigned default/hello-7788754f4b-mrs5f to fault-hunt-control-plane
19s                     Normal    Created                             Pod/hello-7788754f4b-mrs5f      Container created
19s                     Normal    Pulled                              Pod/hello-7788754f4b-mrs5f      Container image "hello:0.1.1" already present on machine and can be accessed by the pod
18s                     Normal    Started                             Pod/hello-7788754f4b-mrs5f      Container started
17s                     Normal    SuccessfulDelete                    ReplicaSet/hello-59fc8dcdfd     Deleted pod: hello-59fc8dcdfd-rmx6z
17s (x5 over 5m58s)     Normal    ScalingReplicaSet                   Deployment/hello                (combined from similar events): Scaled down replica set hello-59fc8dcdfd from 1 to 0

```

After checking the logs of both pods, I see the web server is up and running:

```
kubectl logs pods/hello-7788754f4b-mrs5f
2026/10/27 22:58:05 listening on :8080
robi@debian:~/programming/kubernetes-origoss/kubernetes-and-devops$ kubectl logs pods/hello-7788754f4b-x6747
2026/10/27 22:58:02 listening on :8080


```
This confirms my hunch about the fact that this might be a port exposure / port publishing or other configuration issue inside `service.yaml`. I will open `service.yaml` now.

AAAANNDDD there it is!

At the end of the file I see port: 80 and targetPort: 80. This is wrong because the service should route stuff into our pods. But our containers use port 8080. I will change `targetPort: 80` to `targetPort: 8080` at line 13 inside `service.yaml` and then apply the file and check the connection again.

Upon verifying the service is updated:

```
kubectl get services
NAME         TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE
hello        ClusterIP   10.96.82.60   <none>        80/TCP    4h5m
kubernetes   ClusterIP   10.96.0.1     <none>        443/TCP   4h6m
robi@debian:~/programming/kubernetes-origoss/kubernetes-course-origoss-personal/fault-hunt$ kubectl describe service hello
Name:                     hello
Namespace:                default
Labels:                   app=hello
Annotations:              <none>
Selector:                 app=hello,tier=web
Type:                     ClusterIP
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       10.96.82.60
IPs:                      10.96.82.60
Port:                     <unset>  80/TCP
TargetPort:               8080/TCP
Endpoints:                
Session Affinity:         None
Internal Traffic Policy:  Cluster
Events:                   <none>


```
I checked curl again. Unfortunately, I am getting the same error. This is probably because the port is not published. I will test this by creating a port forward on my laptop like this:

`kubectl port-forward svc/hello 8080:80`

unfortunately, this did not work either. Let's check the endpointslices:

```
kubectl get svc -o wide
NAME         TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE     SELECTOR
hello        ClusterIP   10.96.82.60   <none>        80/TCP    4h18m   app=hello,tier=web
kubernetes   ClusterIP   10.96.0.1     <none>        443/TCP   4h19m   <none>
robi@debian:~/programming/kubernetes-origoss/kubernetes-course-origoss-personal/fault-hunt$ kubectl get endpointslices

| NAME | ADDRESSTYPE | PORTS | ENDPOINTS | AGE |
| :--- | :--- | :--- | :--- | :--- |
| hello-vwdq5 | IPv4 | `<unset>` | `<unset>` | 4h19m |
| kubernetes | IPv4 | 6443 | 172.18.0.3 | 4h20m |


```
The key clue here is `PORTS <unset>`. I believe this might an issue.

Upon checking the service like this:

```
kubectl describe services
Name:                     hello
Namespace:                default
Labels:                   app=hello
Annotations:              <none>
Selector:                 app=hello,tier=web
Type:                     ClusterIP
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       10.96.82.60
IPs:                      10.96.82.60
Port:                     <unset>  80/TCP
TargetPort:               8080/TCP
Endpoints:                
Session Affinity:         None
Internal Traffic Policy:  Cluster


```
I see that we have 2 selectors, and 2 ports. The first port is unset, and the second is 80/tcp. The order of the output makes me think that this means tier=web receives our traffic, and app=hello has no port set. Let's change service.yaml and assign a second port to tier:web. I don't wanna delete tier: web because the requirements says that I should not delete anything, cuz that might mask some other issues.

So I will change service.yaml, and change the end of the file from 

```
ports:
    - port: 80
      targetPort: 8080

```

To 
```
ports:
    - port: 80
      targetPort: 8080
    - port: 81
      targetPort: 8081
```
(I just chose a random port for that other service.)

Update: still not working. It says the property "name" is missing. I added a name for each port, but the connection is still not working.

Of course I noticed that, when compared to the service file inside my session / lecture folder, there is one stark difference: In this file, we have a line that says "tier : web", and if I remove this line, everything works, and the connection reaches the cluster after a port forward. However, I am sure I should not remove this. I feel stuck so I will ask ChatGPT  why that single line breaks everything.

Update: I initially thought that these labels define a logical OR relationship, hence why I was dumbfounded. ChatGPT corrected me and said that spec:selector actually follows an AND logical operation, so my service is trying to find pods that have both app=hello and tier=web as their labels. Of course this won't work, so I either have to remove this label, or add this to my deployment.

Since the homework description says we should not delete any existing lines, I will try adding the extra label inside my deployment.yaml file.
First add it to line 6-7: metadata:labels: app: hello and tier: web. Then line 12-13: spec:selector:matchLabels: app:hello and tier:web
And finally spec:template:metadata:labels on line 17-17: app:hello and tier: web

Upon adding these lines and running apply, I got hit with an error:

```

kubectl apply -f deployment.yaml 
The Deployment "hello" is invalid: spec.selector: Invalid value: {"matchLabels":{"app":"hello","tier":"web"}}: field is immutable


```
I don't know why this happens, so I went and asked chatGPT about what this means.

I learned that these fields are immutable because if they changed, a new deployment would need to be created, leaving the old pods behind, and I would need to manually clean them up. Now I think I should just investigate if the 'tier: web' label is necessary in the service.yaml file, or not. Since adding this label to our deployment.yaml is really complicated, it makes me question if it is needed or not.

I want to see all existing pods, and their labels. If I can confidently say that there are 0 pods in our cluster that rely on that tier=web label, then I will remove it from the service.

I ran these commands:


`get pods --show-labels`

| NAME | READY | STATUS | RESTARTS | AGE | LABELS |
| :--- | :--- | :--- | :--- | :--- | :--- |
| hello-7788754f4b-mrs5f | 1/1 | Running | 1 (41m ago) | 12h | app=hello,pod-template-hash=7788754f4b |
| hello-7788754f4b-x6747 | 1/1 | Running | 1 (41m ago) | 12h | app=hello,pod-template-hash=7788754f4b |

`kubectl get nodes --show-labels`


| NAME | STATUS | ROLES | AGE | VERSION | LABELS |
| :--- | :--- | :--- | :--- | :--- | :--- |
| fault-hunt-control-plane | Ready | control-plane | 16h | v1.37.0 | beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,kubernetes.io/arch=amd64,kubernetes.io/hostname=fault-hunt-control-plane,kubernetes.io/os=linux,node-role.kubernetes.io/control-plane= |



Now I know that there are no pods, nor nodes that rely on the tier=web label, so I will just remove that from service.yaml

Upon removing tier=web from the `service.yaml` file, everything seems to be working. I think it is time to run the verification script.

## Fault 8

- **Symptom** (what you saw first): After running the validation script, I saw 8/8 passed, so I did not explictly see any other faults afterwards. Perhaps I accidentally fixed fault #8 in an earlier fix somewhere above.
- **Command** (that showed the cause):
- **Evidence** (the exact output line):
- **Cause**:
- **Fix** (file and field you changed):
