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
