# Default backend

The default backend is a service which handles all URL paths and hosts the nginx controller doesn't understand (i.e., all the requests that are not mapped with an Ingress).

## Custom behavior

If a request to this service contains the header `x-original-uri` then it will be forwarded to a  service in the defined namespace.

## Endpoints
Basically a default backend exposes two URLs:

- `/healthz` that returns 200
- `/` that returns 404