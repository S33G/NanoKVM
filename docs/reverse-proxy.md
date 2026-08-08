# Reverse Proxy

Terminate public HTTPS at the reverse proxy and proxy NanoKVM at the origin root. NanoKVM does not support a base path: publish `https://kvm.example.com/`, not `https://example.com/kvm/`.

The OIDC callback must remain externally reachable at:

```text
https://kvm.example.com/api/auth/oidc/callback
```

Register that exact URL at the provider and use it as `oidc.redirectUri`. Forward the original `Host` and public scheme. `X-Forwarded-Proto: https` is also used when NanoKVM decides whether its session cookie must be `Secure`.

Restrict direct access to the NanoKVM HTTP listener so clients cannot bypass proxy TLS or supply trusted-looking forwarding headers. The examples include client IP headers for logs and proxy-aware components; only trust values written by your controlled edge proxy.

## Caddy

Replace `192.0.2.10:80` with the NanoKVM address. Caddy obtains and renews the public certificate when DNS and network policy permit it.

```caddyfile
kvm.example.com {
    reverse_proxy http://192.0.2.10:80 {
        header_up Host {host}
        header_up X-Forwarded-Proto {scheme}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}

        flush_interval -1
        transport http {
            read_timeout 0
            write_timeout 0
        }
    }
}
```

Caddy handles WebSocket upgrades automatically. `flush_interval -1` avoids buffering streaming responses, while zero transport read/write timeouts allow long-lived streams and WebSockets. Apply connection limits at the public edge if required rather than imposing a short upstream timeout.

If Caddy itself is behind another proxy, configure Caddy's trusted proxy handling separately and ensure `{scheme}` still represents the public HTTPS request. Do not accept arbitrary client-supplied `X-Forwarded-*` headers at an exposed NanoKVM listener.

## Nginx

Place the `map` in the `http` context and the `server` block in the normal virtual-host configuration. Replace the upstream address and certificate paths.

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name kvm.example.com;

    ssl_certificate     /etc/letsencrypt/live/kvm.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/kvm.example.com/privkey.pem;

    location / {
        proxy_pass http://192.0.2.10:80;
        proxy_http_version 1.1;

        proxy_set_header Host              $http_host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;

        proxy_set_header Upgrade    $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        proxy_buffering off;
        proxy_request_buffering off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}

server {
    listen 80;
    listen [::]:80;
    server_name kvm.example.com;
    return 308 https://$host$request_uri;
}
```

`$http_host` preserves a public non-default port, which NanoKVM's browser WebSocket origin check requires. The trailing slash behavior is intentional: `proxy_pass` has no URI suffix, so `/api/auth/oidc/callback`, API routes, static files, streams, and WebSocket paths are forwarded unchanged. Do not add a location that rewrites NanoKVM beneath another prefix.

Increase the one-hour timeouts if sessions are expected to remain idle longer. Buffering is disabled for MJPEG and other streaming responses. WebSocket upgrades cover keyboard/mouse control, direct H.264, WebRTC signaling, terminal, and other WebSocket-backed features.

## Verification

1. Open `https://kvm.example.com/` and confirm no browser TLS warning appears.
2. Request `https://kvm.example.com/api/auth/config` and confirm the public auth response is returned.
3. Start OIDC login and verify the provider receives the exact HTTPS callback.
4. After login, request `GET /api/auth/session` through the same origin and confirm `authSource` is `oidc`.
5. Exercise the selected video mode and keyboard/mouse control for longer than any default proxy timeout.
6. Confirm the browser stores `nano-kvm-token` as `HttpOnly`, `SameSite=Lax`, and `Secure`.

See [OIDC and Authentik](oidc.md) for provider configuration.
