# Plugin POC

This POC will install Consul Connect, Ambassador, Prometheus, Grafan, and Zipkin, along with some demo services.

## Initial setup

1. If you're on GKE, make sure you have an admin cluster role binding:

```
kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")
```

2. Clone this repository locally:

```
git clone https://github.com/datawire/plugin-poc.git
```

3. Create the `ambassador-pro-registry-credentials` secret in your cluster, if you haven't done so already.

4. Add your license key to the `ambassador-pro.yaml` file.

5. Initialize Helm with the appropriate permissions.

```
kubectl apply -f helm-rbac.yaml
helm init --service-account=tiller
```

## Set up Consul Connect

We will install Consul via Helm.

**Note:** The `values.yaml` file is used to configure the Helm installation. See [documentation](https://www.consul.io/docs/platform/k8s/helm.html#configuration-values-) on different options. We're providing a sample `values.yaml` file.

```shell
helm repo add consul https://consul-helm-charts.storage.googleapis.com
helm install --name=consul consul/consul -f ./consul-connect/values.yaml
```

This will install Consul with Connect enabled. 

**Note:** Sidecar auto-injection is not configured by default and can enabled by setting `connectInject.default: true` in the `values.yaml` file.

## Verify the Consul installation

Verify you Consul installation by accessing the Consul UI. 

```shell
kubectl port-forward service/consul-ui 8500:80
```

Go to http://localhost:8500 from a web-browser.

If the UI loads correctly and you see the consul service, it is safe to assume Consul is installed correctly.

## Install Ambassador

1. Install Ambassador with the following commands, along with the demo QOTM service and a route to HTTPbin service.
   
   ```
   kubectl apply -f statsd-sink.yaml
   kubectl apply -f ambassador-pro.yaml
   kubectl apply -f ambassador-service.yaml
   kubectl apply -f httpbin.yaml
   kubectl apply -f consul-connect/qotm
   ```

2. Get the IP address of Ambassador: `kubectl get svc ambassador`.

3. Send a request to the QOTM service; this request will fail because the request is not properly encrypted.


   ```shell
   curl -v http://{AMBASSADOR_IP}/qotm/

   < HTTP/1.1 503 Service Unavailable
   < content-length: 57
   < content-type: text/plain
   < date: Thu, 21 Feb 2019 16:29:30 GMT
   < server: envoy
   < 
   upstream connect error or disconnect/reset before headers
   ```

4. Send a request to `httpbin`; this request will succeed since this request is sent to a service outside of the service mesh.

   ```
   curl -v http://{AMBASSADOR_IP}/httpbin/ip
   {
      "origin": "108.20.119.124, 35.184.242.212, 108.20.119.124"
   }
   ```

## Consul Connect integration

Now install the Consul Connect integration.

```shell
kubectl apply -f consul-connect/ambassador-consul-connector.yaml
```

### Verify correct installation

Verify that the `ambassador-consul-connect` secret is created. This secret is created by the integration.

```shell
kubectl get secrets

ambassador-consul-connect                                 kubernetes.io/tls                     2     
ambassador-pro-consul-connect-token-j67gs                 kubernetes.io/service-account-token   3     
ambassador-pro-registry-credentials                       kubernetes.io/dockerconfigjson        1     
ambassador-token-xsv9r                                    kubernetes.io/service-account-token   3     
cert-manager-token-tggkd                                  kubernetes.io/service-account-token   3     
consul-connect-injector-webhook-svc-account-token-4xpw9   kubernetes.io/service-account-token   3     
```

You can now send a request to QOTM, which will be encrypted with TLS. Note that we're sending an unencrypted HTTP request, which gets translated to TLS when the request is sent to Consul Connect. (Ambassador also supports TLS encryption, which is beyond the scope of this document.)

```shell
curl -v http://{AMBASSADOR_IP}/qotm/

< HTTP/1.1 200 OK
< content-type: application/json
< content-length: 164
< server: envoy
< date: Thu, 21 Feb 2019 16:30:15 GMT
< x-envoy-upstream-service-time: 129
< 
{"hostname":"qotm-794f5c7665-26bf9","ok":true,"quote":"The last sentence you read is often sensible nonsense.","time":"2019-02-21T16:30:15.572494","version":"1.3"}
```

## Metrics

Next, we'll set up metrics using Prometheus and Grafana.

1. Install the Prometheus Operator.

   ```
   kubectl apply -f monitoring/prometheus.yaml
   ```

2. Wait 30 seconds until the `prometheus-operator` pod is in the `Running` state.

3. Create the rest of the monitoring setup:

   ```
   kubectl apply -f monitoring/prom-cluster.yaml
   kubectl apply -f monitoring/prom-svc.yaml
   kubectl apply -f monitoring/servicemonitor.yaml
   kubectl apply -f monitoring/grafana.yaml
   ```

4. Send some traffic through Ambassador (metrics won't appear until some traffic is sent). You can just run the `curl` command to httpbin above a few times.

5. Get the IP address of Grafana: `kubectl get svc grafana`

6. In your browser, go to the `$GRAFANA_IP` and log in using username `admin`, password `admin`.

7. Configure Prometheus as the Grafana data source. Give it a name, choose type Prometheus, and point the HTTP URL to `http://prometheus.default:9090`. Save & Test the Data Source.

8. Import a dashboard. Click on the + button, and then choose Import. Upload the `ambassador-dashboard.json` file to Grafana. Choose the data source you created in the previous step, and click import.

9. Go to the Ambassador dashboard!

## Distributed Tracing

We will now set up distributed tracing using Zipkin.

1. Create the `TracingService`

    ```
    kubectl apply -f tracing/zipkin.yaml
    kubectl apply -f tracing/tracing-config.yaml
    ```
  
    This will tell Ambassador to generate a tracing header for all requests through it. Ambassador needs to be restarted for this configuration to take effect.

2. Restart Ambassador to reload the Tracing configuration.

    - Get your Ambassador Pod name

       ```
       kubectl get pods

       ambassador-7bbd676d59-7b8w6                                   2/2     Running   0          10m
       ambassador-pro-consul-connect-integration-6d7d489b4b-fwndt    1/1     Running   0          4h
       ambassador-pro-redis-6cbb7dfbb-pzg66                          1/1     Running   0          4h
       consul-76ks4                                                  1/1     Running   0          4h
       consul-connect-injector-webhook-deployment-7846847f9f-r8w8p   1/1     Running   0          4h
       consul-p89q4                                                  1/1     Running   0          4h
       consul-server-0                                               1/1     Running   0          4h
       consul-server-1                                               1/1     Running   0          4h
       consul-server-2                                               1/1     Running   0          4h
       consul-vvl7b                                                  1/1     Running   0          4h
       qotm-794f5c7665-26bf9                                         2/2     Running   0          4h
       zipkin-98f9cbc58-zjksk                                        1/1     Running   0          7m
       ````

    - Deleting the pod will tell Kubernetes to restart another one

       ```
       kubectl delete po ambassador-7bbd676d59-7b8w6 
       pod "ambassador-7bbd676d59-7b8w6" deleted
       ```

    - Wait for the new Pod to come online

3. Apply the micro donuts demo service

    ```
    kubectl apply -f tracing/microdonut.yaml
    ```

4. Test the tracing service

    - From a web-browser, go to http://{AMBASSADOR_IP}/microdonut/
    - Use the UI to select and order a number of donuts
    - After clicking `order`, from a new tab, access http://{AMBASSADOR_IP}/zipkin/
    - In the search parameter box, expand the `Limit` to 1000 so you can see all of the traces
    - Click `Find Traces` and you will see a list of Traces for requests through Ambassador.
    - Find a trace that has > 2 spans and you will see a trace for all the request our donut order made
    
## Dynamic Routing

Now we will configure a filter to dynamically route to different services based on the URL parameter `userID`. 

If you run `kubectl get pods`, you will see we have 4 different instances of the QoTM pod running. 

```shell
$ kubectl get pods
...
qotm-1-7cdb6785d5-kp8g8                                      2/2     Running   0          2h
qotm-2-5dddd9f558-f7msr                                      2/2     Running   0          3h
qotm-3-68b4844c8d-s8bdd                                      2/2     Running   0          3h
qotm-7f48dd8c97-rsfj5                                        2/2     Running   0          1h
...
```

`qotm-1`, `qotm-2`, and `qotm-3` will simulate a request being routed to different versions of an app running in different data centers and `qotm` will act as a fall-back endpoint if the request doesn't belong to any of the three.

`qotm-1`, `-2`, or `-3`are selected by getting the value of the `userID` URL parameter and querying a Consul Key-Value store for the service it should be routed to.

To configure this:

1. Create the Consul KV store with the provided KV-Pairs

   ```shell
   kubectl cp consul-connect/kv.sh consul-server-0:kv.sh
   kubectl exec -it consul-server-0 -- sh kv.sh 
   ```

   This will create the following KV store:

   | userID | DC |
   |--------|----|
   | 1      | 1  |
   | 2      | 2  |
   | 3      | 2  |
   | 4      | 3  |
   | 5      | 1  |
   | 6      | 3  |
   | 7      | 1  |
   | 8      | 2  |
   | 9      | 1  |
   | 10     | 2  |

2. Create the `Filter` and `FilterPolicy` that tells Ambassador to do this key-value lookup

   ```shell
   kubectl apply -f ambassador-pro-auth.yaml
   kubectl apply -f x-dc-filter.yaml
   ```

3. Test the dynamic routing with curl:

   ```
   $ curl http://$AMBASSADOR_IP/qotm/?userID=1

   {"hostname":"qotm-3-68b4844c8d-s8bdd","ok":true,"quote":"QOTM Service 1","time":"2019-03-07T22:52:26.932964","version":"1.3"}

   $ curl http://$AMBASSADOR_IP/qotm/?userID=2

   {"hostname":"qotm-3-68b4844c8d-s8bdd","ok":true,"quote":"QOTM Service 2","time":"2019-03-07T22:52:26.932964","version":"1.3"}

   $ curl http://$AMBASSADOR_IP/qotm/?userID=4

   {"hostname":"qotm-3-68b4844c8d-s8bdd","ok":true,"quote":"QOTM Service 3","time":"2019-03-07T22:52:26.932964","version":"1.3"}
   ```

## Routing to VMs

Ambassador can route to services running on VMs outside of Kubernetes. This can be done by either routing to the DNS name of the service running on the VM or by routing to the IP address using a Kubernetes `Endpoint` `Service`.

The file vm-routing.yaml contains `Mapping`s that can do both.


### DNS

1. Edit the `service:` in the `dns_mapping` `Mapping`.

   ```
         ---
         apiVersion: ambassador/v1
         kind: Mapping
         name: dns_mapping
         prefix: /dns/
         service: httpbin.org
    ```

2. `kubectl apply -f vm-routing.yaml`

3. Send a request to `/dns/` over curl:

   ```
   curl $AMBASSADOR_IP/dns/
   ```

### IP

1. Edit the `ip` and `port`in the `vm-routing` `Endpoint` to point to the IP of your VM

   ```
   subsets:
     - addresses:
         - ip: 34.197.95.106
       ports:
         - port: 80
   ```

2. `kubectl apply -f vm-routing.yaml`

3. Send a request to `/ip-endpoint/` over curl:

   ```
   curl $AMBASSADOR_IP/ip-endpoint/
   ```

**Note:** Both the `ip-endpoint` and `DNS` methods currently route to httpbin.org which is running on a VM outside of Ambassador for demonstration purposes.

## JWT

1. Configure the JWT filter:

   ```
   kubectl apply -f jwt-filter.yaml
   kubectl apply -f ambassador-pro-auth.yaml
   ```

2. Send a valid JWT to the `jwt-httpbin` URL:

   ```
   curl -i --header "Authorization: Bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." $AMBASSADOR_IP/jwt-httpbin/ip
   ```

3. Send an invalid JWT, and get a 401:

   ```
   curl -i $AMBASSADOR_IP/jwt-httpbin/ip
   HTTP/1.1 401 Unauthorized
   content-length: 58
   content-type: text/plain
   date: Thu, 28 Feb 2019 01:07:10 GMT
   server: envoy
   ```

4. Note that we've configured the `jwt-httpbin` URL to require JWTs, but the `httpbin` URL does not:

   ```
   curl -v http://$AMBASSADOR_IP/httpbin/ip
   {
      "origin": "108.20.119.124, 35.184.242.212, 108.20.119.124"
   }
   ```

The JWT is validated using public keys supplied in a JWKS file. For the purposes of this demo, we're supplying a Datawire JWKS file. You can change the JWKS file by modifying the `filter.yaml` manifest and changing the `jwksURI` value.

## Websockets

1. Create the websockets service:

   ```
   kubectl apply -f websockets/ws_server.yaml
   ```

   This creates a websockets server running in Kubernetes. It also creates an Ambassador `Mapping` for routing websockets connections to the `prefix: /ws_sync/`.

2. Add your $AMBASSADOR_IP to the HTML client (`websockets/client.html`):

   ```html
    50.        <script>
    51.            var minus = document.querySelector('.minus'),
    52.                plus = document.querySelector('.plus'),
    53.                value = document.querySelector('.value'),
    54.                users = document.querySelector('.users'),
    55.                websocket = new WebSocket("ws://{AMBASSADOR_IP}/ws_sync/");
   ```

3. Open the HTML client in your favorite web browser.

   This simple client increments or decrements a counter on the server. This counter is synchronized across all clients.

## Key Takeaways

* We're dynamically registering and resolving routes for services, e.g., in `consul-connect/qotm.yaml` and `httpbin.yaml`, new services are dynamically registered with Ambassador. Ambassador then uses Kubernetes DNS to resolve the actual IP address of these services.
* Ambassador is processing inbound (North/South) requests to the mesh, and dynamically processing URL variables to determine where a given request should be routed.
* Ambassador automatically obtains certificates from Consul Connect, and uses these certificate to originate encrypted TLS connections to target services in the mesh.
* Prometheus is collecting high resolution metrics from Ambassador. A sample dashboard of some of these metrics is displayed in Grafana.

### More about the Custom Filter

* A Golang plugin is looking at the request ID, and setting an HTTP header called `X-Dc` to Odd or Even
* In the `httpbin.yaml`, we create several mappings. One mapping maps to `X-Dc: Odd` and routes to QoTM. The other is a mapping that routes to microdonuts.
* The source code to the Golang plugin is in https://github.com/datawire/apro-example-plugin (look at `param-plugin.go`).
* Updating the plug-in involves making changes to the source code, `make DOCKER_REGISTRY=...`, and then updating the `ambassador-pro.yaml` sidecar to point to the new image.
