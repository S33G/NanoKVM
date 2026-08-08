# Native OIDC Authentication

## Summary

Add optional native OIDC login alongside the existing local NanoKVM account. OIDC identities are session-only; provider tokens are not retained and local users are not created.

## Implementation Plan

- Add `oidc` configuration with secure defaults and `clientSecretFile` support.
- Discover provider metadata and use authorization code flow, PKCE S256, state, and nonce.
- Validate ID tokens and configured claims, then apply verified-email and allowed-group admission policy.
- Issue the existing NanoKVM session cookie and expose public auth configuration plus session metadata to the UI.
- Keep local login enabled by default and reject OIDC-sourced sessions when OIDC is disabled.
- Document Authentik, reverse proxies, API authentication, recovery, upgrades, and limitations.

## Security Review

- HTTPS is mandatory for issuer and callback URLs except on loopback.
- Client secret sources are mutually exclusive; secret files must be non-empty `0600` regular files.
- Login transactions are one-time, bounded, and bound to a short-lived HttpOnly browser cookie, with state, nonce, and PKCE validation.
- Issuer, audience, authorized party, signature, expiry, subject, and nonce are validated through OIDC ID token verification.
- Provider tokens are used only during callback processing and are not retained.
- Browser WebSocket upgrades enforce a same-host origin check in addition to the existing NanoKVM session check.
- Sessions use HttpOnly, SameSite=Lax cookies; the application still has no general-purpose CSRF-token mechanism.
- `allowedGroups` controls admission. `adminGroups` is metadata only and is explicitly not an authorization boundary.
- All admitted users retain full existing NanoKVM privileges because role enforcement is outside this change.
- Recovery keeps local login enabled by default; disabling global authentication remains an emergency-only bypass.

## Compatibility

- Existing configuration remains valid; OIDC defaults to disabled.
- `authentication` remains `enable` or `disable`.
- Existing local accounts and login payloads remain supported unless `allowLocalLogin` is explicitly false; successful local login now also sets the hardened server cookie.
- The existing `nano-kvm-token` cookie remains the credential for APIs, streams, and WebSockets.
- Root-path deployments are supported; URL base paths are not.

## Tests

- [x] Unit tests: configuration validation and secret-file permissions
- [x] Unit tests: state, nonce, PKCE, and transaction replay/expiry
- [x] Unit tests: claim fallback, group admission, admin metadata, and verified email
- [x] Integration tests: discovery, callback, ID token validation, session creation, and protected middleware access
- [x] Frontend tests: OIDC visibility/errors, local fallback, session gating, and logout
- [ ] Manual proxy test: Caddy and Nginx HTTPS callback, cookies, streams, and WebSockets
- [ ] Manual test: Authentik confidential provider and mapped group claims

Automated validation performed:

```sh
# In a Go 1.25 container
go test ./service/auth ./middleware
go test -race ./service/auth ./middleware
go vet ./...

# In a Node 22 / pnpm 11 container
pnpm test
pnpm lint
pnpm build

# With the repository's RISC-V musl compiler mounted into Go 1.25
CGO_ENABLED=1 GOOS=linux GOARCH=riscv64 \
  CC=riscv64-unknown-linux-musl-gcc \
  CGO_CFLAGS="-mcpu=c906fdv -march=rv64imafdcv0p7xthead -mcmodel=medany -mabi=lp64d" \
  go build -o /tmp/NanoKVM-Server .
```

`go test ./...` passes all compilable host packages, but host linking fails for `service/picoclaw` and `service/ws` because this checkout contains only the RISC-V `server/dl_lib/libkvm.so`. The RISC-V CGO server build succeeds. The full release-builder image could not be exported within the available timeout, so `make release-build` and a complete firmware package could not be completed; `make package VERSION=0.0.0` correctly stopped because `server/NanoKVM-Server` was not staged by that release build.

## Limitations

- No local OIDC user records or automatic user creation.
- No provider token retention, refresh, revocation, or provider logout.
- No bearer-token or non-interactive OIDC API flow.
- No role enforcement; `adminGroups` only annotates session metadata.
- Group changes are evaluated on the next login, not continuously.
- No base-path deployment support.

## Files

- Backend configuration, session, routing, OIDC service, and tests under `server/config`, `server/middleware`, `server/proto`, `server/router`, and `server/service/auth`
- WebSocket origin integration in the existing stream, terminal, HID socket, and PicoClaw gateway services
- Frontend authentication UI/API/state, all locale catalogs, Vitest configuration/tests, and pnpm manifests under `web`
- `docs/oidc.md`, `docs/reverse-proxy.md`, and `docs/api-authentication.md`
- Root/server README links and `CHANGELOG.md`
