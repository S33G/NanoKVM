# API Authentication

This document describes authentication behavior for the current NanoKVM HTTP service. It is not a complete or versioned API specification.

## Sessions

With `authentication: enable`, successful local or OIDC login creates a NanoKVM session. The server signs a session token and sets it in this cookie:

| Attribute | Value |
| --- | --- |
| Name | `nano-kvm-token` |
| Path | `/` |
| HttpOnly | yes |
| SameSite | `Lax` |
| Secure | yes on HTTPS deployments, including when a trusted reverse proxy reports `X-Forwarded-Proto: https` |
| Lifetime | `jwt.refreshTokenDuration` |

The browser sends this cookie to ordinary APIs, HTTP streams, and WebSocket handshakes. NanoKVM does not use an OIDC access token as an API bearer token and does not retain provider access, refresh, or ID tokens. There is no documented `Authorization: Bearer` equivalent for the web session cookie.

OIDC identities exist only in the signed NanoKVM session. They are not persisted as local accounts. All admitted local and OIDC sessions currently have the full existing NanoKVM privileges; the `admin` session field is metadata and is not enforced as an authorization boundary.

When `authentication: disable`, normal token checks are bypassed and OIDC login is suppressed because no login is required. This exposes the device's existing capabilities and is not an OIDC mode.

## Authentication Routes

| Method and path | Authentication | Behavior |
| --- | --- | --- |
| `GET /api/auth/config` | Public | Returns login UI configuration: OIDC enabled/readiness state, a non-secret error code, provider label, and local-login availability. |
| `POST /api/auth/login` | Public | Performs existing local account login and sets the session cookie. Returns an application error if OIDC disables local login. |
| `GET /api/auth/oidc/login` | Public | Starts provider discovery and an authorization-code flow with PKCE S256, browser-bound state, and nonce, then redirects to the provider. |
| `GET /api/auth/oidc/callback` | Public | Consumes the one-time login transaction, exchanges the code, validates the ID token and claims, applies admission policy, sets the NanoKVM session cookie, and redirects to the root UI. |
| `GET /api/auth/session` | Public | Returns `authenticated: false` without a valid cookie, or the current NanoKVM identity metadata with one. It does not query the provider. |
| `POST /api/auth/logout` | Session cookie | Clears the cookie. If `jwt.revokeTokensOnLogout` is true, rotating the signing secret also invalidates all existing NanoKVM sessions. It does not log the browser out of the OIDC provider. |

Existing protected account routes also remain under `/api/auth`, including `/api/auth/account` and `/api/auth/password`.

### Response Examples

NanoKVM's ordinary JSON handlers generally use an application envelope with HTTP 200:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "oidcEnabled": true,
    "oidcReady": true,
    "providerName": "Authentik",
    "allowLocalLogin": true
  }
}
```

An OIDC session response has this shape inside the same envelope:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "authenticated": true,
    "username": "alice",
    "displayName": "Alice Example",
    "email": "alice@example.com",
    "authSource": "oidc",
    "admin": false
  }
}
```

`authSource` can be `local`, `oidc`, or `disabled`. For OIDC, `admin` reflects an `adminGroups` match only; it does not alter access.

## Protected Route Families

The session middleware is used by the browser-facing route families under `/api`, including:

- `/api/application`
- `/api/auth` protected operations
- `/api/vm`
- `/api/stream`
- `/api/storage`
- `/api/network` protected operations
- `/api/hid`
- `/api/download`
- `/api/extensions`
- `/api/ai/control`
- `/api/picoclaw` browser-facing operations
- `/api/mcp` management operations
- `/api/ws`

This list describes route families rather than promising every method, request body, or response as a stable public API. Some routes have intentionally different security models. Examples include public AP-mode Wi-Fi setup routes, loopback-only internal PicoClaw/HID routes, and the MCP protocol endpoint protected by its own API key. Consult the current router implementation before automating a non-browser endpoint.

HTTP streams and WebSocket upgrades authenticate the initial request with `nano-kvm-token`. Browser WebSocket requests must also have an `Origin` host matching the request `Host`; non-browser clients without `Origin` remain supported. A reverse proxy must preserve `Host`, forward cookies, preserve paths, support WebSocket upgrade headers, and avoid buffering or short timeouts for streams.

## Errors

| Condition | Result |
| --- | --- |
| OIDC is enabled but its static configuration is invalid | `GET /api/auth/config` returns `oidcReady: false` and `oidcError: "invalid_configuration"` in a successful envelope. Readiness does not test live provider reachability. |
| Missing, expired, invalid, or otherwise rejected session on a protected route | HTTP `401` with JSON string `"unauthorized"` in the common middleware path. |
| OIDC-sourced session while `oidc.enabled` is false | HTTP `401`, even if the NanoKVM JWT has not expired. |
| Local login input or credentials fail | HTTP 200 application envelope with nonzero `code` and a `msg`; callers must inspect both HTTP status and envelope code. |
| Local login is disabled | HTTP 200 envelope with `code: -6` and `msg: "local login is disabled"`. |
| OIDC browser flow fails | HTTP 303 redirect to `/#/auth/login?oidc_error=<code>`. |

OIDC browser error codes are `oidc_disabled`, `oidc_invalid_config`, `oidc_provider_unavailable`, `oidc_access_denied`, `oidc_rate_limited`, and `oidc_callback_failed`. They are deliberately coarse; use server and provider logs for diagnosis without exposing token or claim details to the browser.

## Programmatic Use

- Use a cookie jar and send `nano-kvm-token` on subsequent HTTPS requests. Do not depend on reading it from browser JavaScript because it is `HttpOnly`.
- OIDC login is an interactive redirect flow. NanoKVM has no client-credentials, device-code, resource-owner-password, token exchange, refresh-token API, or API-token minting endpoint for OIDC users.
- The local login payload follows the existing NanoKVM web client's password encoding and is not a general plain-password API contract. Reuse of that internal scheme by external clients is not a stable integration surface.
- Do not send Authentik access or ID tokens to NanoKVM APIs; they are not accepted as bearer credentials.
- Cookie authentication and `SameSite=Lax` are designed for same-origin browser use. NanoKVM does not document a general cross-origin API or CORS contract.
- NanoKVM does not currently issue general-purpose CSRF tokens. `SameSite=Lax` and same-host WebSocket origin checks are part of the browser security boundary, so do not weaken or rewrite these cookie/origin properties at a proxy.
- Logout may invalidate every NanoKVM web session when signing-key rotation is enabled. It does not revoke provider tokens because NanoKVM does not retain them.
- For non-browser automation, use a feature's documented dedicated credential mechanism when one exists, such as the separately configured MCP API key, rather than scripting OIDC.

See [OIDC and Authentik](oidc.md) and [Reverse Proxy](reverse-proxy.md).
