# Testing the Auth Header Transform Plugin

## Start the services

```bash
docker compose up -d traefik
```

## Test the middleware

Send a request with an Authorization header:

```bash
curl -v http://localhost/auth/health \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"
```

The Traefik middleware will:

1. Strip the `Authorization` header
2. Add `X-Auth-Claims` with base64-encoded token
3. Add `X-Auth-Ts` with current UNIX timestamp
4. Add `X-Auth-Sig` with HMAC-SHA256 signature

## Verify the headers reach your backend

Check your auth-service logs to see the transformed headers:

```bash
docker logs -f auth_service
```

You should see the incoming request has:

- ❌ No `Authorization` header
- ✅ `X-Auth-Claims` header
- ✅ `X-Auth-Ts` header
- ✅ `X-Auth-Sig` header

## Debug the plugin

Check Traefik logs:

```bash
docker logs -f traefik
```

If the plugin fails to load, you'll see errors about the AUTH_SHARED_SECRET or module loading.
