# Traefik Auth Header Transform Plugin

This Traefik middleware plugin transforms the `Authorization` header into three signed headers:

- `X-Auth-Claims`: Base64-encoded substring of the Authorization header (after stripping prefix)
- `X-Auth-Ts`: UNIX timestamp (seconds since epoch)
- `X-Auth-Sig`: HMAC-SHA256 signature of `{timestamp}.{claims}` using a shared secret

The original `Authorization` header is stripped from the request.

## Configuration

```yaml
http:
  middlewares:
    auth-header-transform:
      plugin:
        authheader:
          sharedSecretEnvVar: "AUTH_SHARED_SECRET"  # default
          authHeaderPrefix: "Bearer "                # default
```

## Environment Variables

- `AUTH_SHARED_SECRET`: The shared secret used for HMAC signing (required)

## Usage

The plugin is loaded as a local plugin by mounting the plugin directory into the Traefik container.
