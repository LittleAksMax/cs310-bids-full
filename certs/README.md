# Certificates

In order to properly replicate a production environment, we can use locally-trusted SSL certificates to enable HTTPS.

## Generating Certificates

We will use `mkcert` to generate our certificates.

The development machine is a Mac, so we run the following commands:

```bash
# Install mkcert if you haven't already
brew install mkcert

# Install certutil with this command (allows CA to be automatically installed in Firefox)
brew install nss

# Install the local CA
mkcert -install

# Generate certificates for your domains
mkcert localhost 127.0.0.1 ::1
```

The above command should generate keys. Make sure to move the generated keys (should be named `localhost+2-key.pem` and `localhost+2.pem` to this directory -- `./certs/`).

## Making Use of Certificates

Since we are using an API Gateway, we have to ensure we enable TLS. We will also redirect all HTTP traffic to HTTPS.

```yaml
entryPoints:
  web:
    address: ":80"
    http:
      redirections: # redirect HTTP traffic to secure entrypoint using HTTPS
        entryPoint:
          to: websecure
          scheme: https
          permanent: true # 301 redirect
  websecure:
    address: ":443"
    http:
      tls: {} # Enable TLS globally for all routers on this entrypoint

# TLS configuration
tls:
  certificates:
    - certFile: /etc/traefik/certs/localhost+2.pem
      keyFile: /etc/traefik/certs/localhost+2-key.pem
```

In our `compose.yml`, we will want to mount the `./certs` directory where the above configurations expect them. We will also want to expose/forward the relevant port for HTTPS.

```yaml
services:
  traefik:
    ports:
      - "80:80"
      - "443:443" # HTTPS port
      - "8080:8080"
    volumes:
      - ./traefik/traefik.yml:/etc/traefik/traefik.yml
      - ./traefik/dynamic:/etc/traefik/dynamic
      - ./certs:/etc/traefik/certs # Mount your mkcert certificates
```

Finally, we make sure that all our routers take the newly defined `websecure` entrypoint:

```yaml

```
