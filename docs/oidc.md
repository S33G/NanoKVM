# OIDC and Authentik

## Overview

NanoKVM can authenticate browser users with an OpenID Connect (OIDC) provider while retaining the existing local account for recovery. OIDC is disabled by default. The top-level `authentication` setting remains `enable` or `disable`; OIDC is configured separately under `oidc`.

OIDC is available only in firmware/application builds that include this change; it is not present in the tagged 2.5.0 application. A supporting build exposes `GET /api/auth/config` with the OIDC fields described below. Check the release notes for the first published version containing this feature rather than copying this configuration onto an older binary.

The login flow uses provider discovery, the authorization code flow, PKCE S256, a one-time state bound to the initiating browser, nonce validation, and ID token validation. NanoKVM may supplement missing ID token claims from the provider's UserInfo endpoint. It then issues its own session in the `nano-kvm-token` cookie. Provider access, refresh, and ID tokens are not retained.

OIDC identities are session-only. NanoKVM does not create or synchronize local users, and there is no `autoCreateUsers` setting. Logging in through OIDC does not change the local account or password.

> **Authorization limitation:** every admitted user currently receives the full existing NanoKVM privileges. `allowedGroups` is a login admission filter. `adminGroups` only sets the `admin` value in session metadata; it does not grant, restrict, or enforce access and **must not be treated as an authorization boundary**.

OIDC issuer and redirect URLs must use HTTPS. Plain HTTP is accepted only when the URL host is loopback, such as `localhost`, `127.0.0.1`, or `::1`. NanoKVM supports deployment at `/` only; URL base paths such as `https://example.com/kvm/` are unsupported.

An IP-KVM exposes video, keyboard, mouse, media, power, networking, firmware controls, and a root terminal. Put NanoKVM and its management proxy on an isolated administration network, restrict ingress, use trusted HTTPS certificates, keep firmware current, and require strong IdP policy. OIDC improves login integration but does not make a publicly exposed KVM safe by itself.

## Configuration

Edit `/etc/kvm/server.yaml`. This complete example keeps local recovery enabled and reads the client secret from a protected file:

```yaml
authentication: enable

oidc:
    enabled: true
    providerName: Authentik
    issuer: https://auth.example.com/application/o/nanokvm/
    clientId: replace-with-client-id
    clientSecretFile: /etc/kvm/oidc-client-secret
    redirectUri: https://kvm.example.com/api/auth/oidc/callback
    scopes:
        - openid
        - profile
        - email
        - groups
    usernameClaim: preferred_username
    usernameFallbackClaims:
        - email
        - name
    displayNameClaim: name
    emailClaim: email
    groupsClaim: groups
    allowedGroups:
        - nanokvm-users
        - nanokvm-admins
    adminGroups:
        - nanokvm-admins
    allowLocalLogin: true
    requireEmailVerified: false
```

Create the recommended secret file as a regular file readable only by its owner:

```sh
install -m 0600 /dev/null /etc/kvm/oidc-client-secret
```

Place only the client secret in that file, without quotes. A trailing newline is ignored. NanoKVM rejects a secret file accessible by group or other users. Use either `clientSecretFile` or `clientSecret`, never both. An inline secret is supported but is less suitable for configuration backups and reviews:

```yaml
oidc:
    clientSecret: replace-with-client-secret
```

When an inline secret is enabled, NanoKVM also requires `/etc/kvm/server.yaml` itself to be a non-symlink regular file without group or other permissions. Startup restricts an existing configuration to mode `0600`; OIDC remains unavailable if those permissions cannot be verified.

Restart NanoKVM after changing the server configuration:

```sh
/etc/init.d/S95nanokvm restart
```

### Reference

| Key | Default | Description |
| --- | --- | --- |
| `authentication` | `enable` | Global HTTP, API, stream, and WebSocket authentication. Valid values remain `enable` and `disable`. Disabling it bypasses authentication and suppresses OIDC login because no login is required. |
| `oidc.enabled` | `false` | Advertise and accept OIDC login. Disabling it also rejects existing OIDC-sourced NanoKVM sessions. |
| `oidc.providerName` | `OIDC` | Provider label displayed by the login UI. |
| `oidc.issuer` | none | Exact OIDC issuer URL without a query or fragment. NanoKVM discovers provider metadata from this issuer. HTTPS is required except on loopback. |
| `oidc.clientId` | none | Confidential OIDC client identifier. |
| `oidc.clientSecret` | none | Inline confidential client secret. Mutually exclusive with `clientSecretFile`. |
| `oidc.clientSecretFile` | none | Recommended path to a non-empty `0600` regular file containing the client secret. |
| `oidc.redirectUri` | none | Exact public callback URL registered at the provider. HTTPS is required except on loopback. For a normal deployment it is `https://<NanoKVM-host>/api/auth/oidc/callback`. |
| `oidc.scopes` | `[openid, profile, email]` | Scopes requested from the provider. `openid` is mandatory. Add a provider-specific groups scope only when its claim mapping requires one. |
| `oidc.usernameClaim` | `preferred_username` | Preferred string claim for the NanoKVM session username. |
| `oidc.usernameFallbackClaims` | `[email, name]` | Ordered string claims used when `usernameClaim` is missing or empty. The stable OIDC `sub` is the final fallback. |
| `oidc.displayNameClaim` | `name` | Optional display-name claim returned in NanoKVM session metadata. It does not change authorization. |
| `oidc.emailClaim` | `email` | Optional email claim returned in session metadata and checked when verified email is required. |
| `oidc.groupsClaim` | `groups` | Claim containing one group string or an array of non-empty group strings. Matching is exact and case-sensitive. |
| `oidc.allowedGroups` | `[]` | Admission allowlist. Empty admits any otherwise valid OIDC identity. A non-empty list requires at least one exact group match. |
| `oidc.adminGroups` | `[]` | Groups that set session metadata `admin: true`. This metadata is not enforced and is not an authorization boundary. |
| `oidc.allowLocalLogin` | `true` | Keep `POST /api/auth/login` available. Leave enabled for recovery until OIDC is tested. |
| `oidc.requireEmailVerified` | `false` | Require a non-empty configured email claim and boolean `email_verified: true`. Missing, string-valued, or false verification is denied. |

## Authentik Setup

The following values are exact protocol values; Authentik menu placement can vary between releases.

1. Open the Authentik administration interface and create an **OAuth2/OpenID Provider** under **Applications > Providers**.
2. Give the provider a name such as `NanoKVM`, select an authorization flow appropriate for your policy, and set the client type to **Confidential**. Record the generated client ID and client secret.
3. Add one strict redirect URI equal to NanoKVM's public callback, for example `https://kvm.example.com/api/auth/oidc/callback`. Do not add a trailing slash, wildcard, base path, or internal device address.
4. Select a signing key so Authentik issues signed ID tokens. Keep the standard `openid`, `profile`, and `email` scope mappings assigned. Add a `groups` scope/property mapping when group admission is enabled, and make sure required claims are included in the ID token or available from UserInfo.
5. Create an Authentik application under **Applications > Applications**, give it a slug such as `nanokvm`, and attach the provider.
6. In the provider details, copy the **OpenID Configuration Issuer** value into `oidc.issuer`. For an application slug of `nanokvm`, it is normally `https://auth.example.com/application/o/nanokvm/`. Use the published value rather than constructing one from an authorization or token endpoint.
7. Create Authentik groups named `nanokvm-users` and `nanokvm-admins`.
8. Assign ordinary users to `nanokvm-users`. Assign administrators to `nanokvm-admins`; include administrators in `nanokvm-users` as well, or list both groups under NanoKVM `allowedGroups` as in the example.
9. Configure `groupsClaim: groups`, map `allowedGroups` to the admitted groups, and map `adminGroups` to `nanokvm-admins`. Remember that `adminGroups` is metadata only in this release and does not reduce or expand device privileges.
10. Put the client ID and secret into NanoKVM, configure the same public callback as `redirectUri`, include `openid`, `profile`, `email`, and any custom groups scope, then restart NanoKVM.
11. Keep an existing local session open and test an ordinary user, an administrator, and a user in neither allowed group in separate private windows. The first two should be admitted, the unauthorized user should see access denied, and only the administrator session should report `admin: true`.
12. If testing fails, compare the exact redirect URI and issuer, inspect the emitted group claim and scope/property mappings, and confirm that NanoKVM and Authentik clocks are synchronized. The troubleshooting table below covers these common failures.

### Authentik Claims

Authentik's standard scope mappings commonly provide `preferred_username`, `name`, `email`, `email_verified`, and group information, but emitted claims depend on the mappings assigned to the provider and on Authentik policy. Inspect the provider's token/claim preview or a test ID token instead of assuming a claim exists.

If `groups` is absent, add an Authentik scope/property mapping that returns the user's group names as a JSON array of strings and assign it to the NanoKVM provider. Use `groups` as the claim name, or set `groupsClaim` to the emitted name. If the mapping is attached to a custom scope, also add that scope to `oidc.scopes`.

Group matching uses the emitted group names exactly, including case and spaces. With a non-empty `allowedGroups`, a missing or malformed groups claim denies login. Authentik group membership changes take effect on the next NanoKVM login; existing NanoKVM sessions are not continuously re-evaluated.

When `requireEmailVerified` is enabled, Authentik must emit both the configured email claim and `email_verified` as a JSON boolean. A textual value such as `"true"` is not accepted.

## Migration and Recovery

1. Back up `/etc/kvm/server.yaml`, `/etc/kvm/pwd`, and the OIDC client secret file before migration.
2. Register the public root-path callback at the provider and configure NanoKVM with `allowLocalLogin: true`.
3. Restart NanoKVM and verify `GET /api/auth/config` reports OIDC as enabled and ready. Readiness validates local configuration; the first login also verifies provider discovery and reachability.
4. Test OIDC in a separate private browser window. Verify the expected username and source with `GET /api/auth/session`.
5. Keep local login enabled unless operational policy explicitly requires otherwise. It is the default recovery path when the provider, DNS, certificates, or network is unavailable.
6. If local login must be hidden, set `allowLocalLogin: false` only after testing console or SSH access and the configuration rollback procedure.

To recover from an OIDC outage, use the still-enabled local login. If local login was disabled, use device console or SSH to set `oidc.allowLocalLogin: true` or `oidc.enabled: false`, then restart the service. As a last-resort maintenance measure, `authentication: disable` bypasses all normal authentication and exposes full NanoKVM control; use it only on an isolated network, restore `enable` immediately, and restart again.

## Upgrade and Backup Security

- Existing installations remain local-login installations because OIDC defaults to disabled.
- Preserve `/etc/kvm/server.yaml`, `/etc/kvm/pwd`, and any `clientSecretFile` across upgrades. Also preserve the provider-side application, redirect URI, claim mappings, and group policy.
- Treat `server.yaml`, the client secret file, and backups containing either as secrets. Restrict access, encrypt off-device backups, and do not commit them to source control.
- Restore secret files as regular files with mode `0600`. Verify the restored external hostname and callback before enabling OIDC.
- Rotate the provider client secret after suspected disclosure and update the file atomically during a maintenance window.
- NanoKVM does not retain provider tokens or maintain an OIDC user database, so there are no provider tokens or OIDC users to back up.
- Logging out can rotate NanoKVM's JWT signing secret when `jwt.revokeTokensOnLogout` is enabled, invalidating all NanoKVM sessions. Provider sessions remain under provider control.

## Troubleshooting

| Symptom | Likely cause | Resolution |
| --- | --- | --- |
| OIDC button is absent | `oidc.enabled` is false, or the UI cannot read public auth configuration | Check `/etc/kvm/server.yaml`, restart the service, and request `GET /api/auth/config`. |
| OIDC is enabled but not ready | Required value is missing, URLs are invalid, `openid` is absent, both secret options are set, or the secret file is invalid | Check `oidcError` from `GET /api/auth/config` and the server log. Use one non-empty secret source and mode `0600` for a regular secret file. |
| Invalid issuer | The Authentik application/provider issuer is wrong or contains a query | Copy the exact issuer from Authentik's discovery details; do not use its authorization or token endpoint. |
| `oidc_provider_unavailable` | Discovery failed because of DNS, routing, TLS trust, issuer mismatch, or provider outage | Fetch the issuer's discovery document from the NanoKVM network, verify its TLS chain and published issuer, then retry. Discovery failures are briefly cached. |
| Provider reports redirect URI mismatch | Authentik and NanoKVM callback values differ | Make both values exactly `https://<host>/api/auth/oidc/callback`, including scheme, host, port, path, and slash handling. |
| Browser loops back to login after callback | Callback cookie was not stored or forwarded, callback reached another host, or proxy scheme information is wrong | Use HTTPS, forward `Host` and `X-Forwarded-Proto`, do not strip `/api/auth/oidc/callback`, and confirm cookies for the public host. |
| `oidc_callback_failed` | State/nonce expired or was reused, code exchange failed, PKCE failed, or ID token validation failed | Start a new login in one tab, check clock synchronization, client credentials, issuer, callback, and provider logs. |
| `oidc_rate_limited` | More than 20 OIDC initiations reached NanoKVM from one direct peer in one minute | Wait one minute and retry. Behind a reverse proxy, this limit is shared by users reaching NanoKVM through that proxy address. |
| `oidc_access_denied` | Provider denied consent, group admission failed, verified email is missing, or required claims are malformed | Inspect emitted claims and compare exact group names. Check `allowedGroups`, `groupsClaim`, `emailClaim`, and `requireEmailVerified`. |
| Groups are missing | Authentik did not include the configured group claim in the ID token or UserInfo response | Configure the groups scope/property mapping, assign it to the provider, and request its scope if required. |
| User is denied | The user is not in any configured `allowedGroups` value | Assign the user to an admitted group or correct the exact case-sensitive mapping. |
| User is admitted with an unexpected username | Preferred and fallback claims are absent or empty | Inspect `preferred_username`, `email`, and `name`; configure claim names deliberately. NanoKVM finally falls back to `sub`. |
| `admin` is false or has no effect | No exact `adminGroups` match, or role enforcement was expected | Correct the group claim if metadata matters. Regardless of its value, all admitted users currently have the same full privileges. |
| OIDC worked before configuration was disabled | OIDC-sourced sessions are rejected while `oidc.enabled` is false | Re-enable a valid OIDC configuration or log in locally. |
| Login succeeds but an API returns 401 | The NanoKVM session cookie was not stored or forwarded, or OIDC was disabled after login | Inspect the cookie on the public host, preserve proxy cookies and scheme/host headers, and request `GET /api/auth/session`. |
| WebSocket connection fails | The proxy did not forward upgrade headers or preserve `Host`, or the browser origin does not match that host | Use the proxy examples, including `$http_host` for Nginx, and preserve cookies and WebSocket upgrades. |
| Streams or keyboard control fail behind a proxy | WebSocket upgrade, buffering, cookie forwarding, or proxy timeout is incorrect | Use the root-path examples in [Reverse Proxy](reverse-proxy.md), disable response buffering, and allow long-lived connections. |
| Local recovery is unavailable | The IdP or group mapping failed after `allowLocalLogin` was disabled | Use console or SSH to restore `oidc.allowLocalLogin: true`, restart NanoKVM, and avoid `authentication: disable` except on an isolated emergency network. |
| Deployment under `/kvm/` fails | Base-path deployment is unsupported | Serve NanoKVM on its own origin at `/`. |

See [API Authentication](api-authentication.md) for route behavior and [Reverse Proxy](reverse-proxy.md) for HTTPS proxy examples.
