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
   kubectl apply -f consul-connect/qotm.yaml
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

We'll now set up a custom Filter that will dynamically route between the QOTM service and micro donuts.

1. We'll first deploy the custom Filter.

   ```
   kubectl apply -f dynamic-route.yaml
   kubectl apply -f ambassador-pro-auth.yaml
   ```

2. `curl` to the load balancer to test different variables:

   ```
   curl $IP/test/?db=1
   curl $IP/test/?db=2
   ```

   Note that if the value is odd, you will go to QoTM service (you'll see a JSON blob). If the value is even, you will go to the microdonuts application (you'll see a stream of HTML). Note that both of these services are in the service mesh.

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
