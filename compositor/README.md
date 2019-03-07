# Compositor

The Compositor is an independent service that takes an incoming request and splits the incoming request to multiple backend services. The responses from these backend services are then aggregated and returned to the client.

For performance and simplicity, the Compositor is written in Golang. The Compositor has a customizable request splitter & response aggregator. In the future, the request splitter & response aggregator could be delegated to a Python program (they are currently part of the core Go program).

## Setup

To deploy the Compositor with Ambassador, run:

`kubectl apply -f compositor.yaml`

This will deploy the Compositor with Ambassador.

## Testing

The Compositor is configured by default to send requests to two backend services: `httpbin` and `qotm`. Send a request to Compositor:

`curl http://$AMBASSADOR_IP/compositor/1`


Note that the Compositor is taking a URL variable of `1` and passing it on to both the httpbin and QOTM services. You can change this value to see what happens.

## Configuring Compositor

The Compositor `SplitRequest` function configures the target backend services. The `JoinResponses` function configures how the responses are aggregated. Both of these functions are designed to be modified by the end user as needed. The following workflow should allow you to make changes to Compositor:

1. Make the appropriate changes to the function(s) in `compositor.go`.

2. Build the changes by typing `make DOCKER_REGISTRY=<your dockery registry>`. 

3. Push the changes to your Docker repository: `docker push ...`.

4. Update the `compositor.yaml` to point to your Docker image.
