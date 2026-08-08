# Mocked Web Development Plan

## Objective

Make `pnpm mocked` a comprehensive, hardware-free environment for developing and smoke-testing the NanoKVM Web UI.

The environment should model the application's practical UI behavior without attempting to emulate hardware media codecs or full WebRTC signaling.

## Scope

The mocked environment will provide:

- Every REST endpoint used by `web/src/api`.
- Authentication configuration, local login, OIDC login presentation, session, logout, and password lifecycle behavior.
- Stateful device settings, virtual media, scripts, downloads, networking, HID, application updates, Tailscale, MCP, and PicoClaw fixtures.
- Browser-intercepted WebSockets for HID/status traffic, the terminal, and the PicoClaw gateway.
- Stateful uploads and mutations that remain visible to subsequent requests.
- A static desktop image in place of WebRTC, Direct H.264, or an actual MJPEG capture source.
- Developer controls for scenario selection, reset, global latency, endpoint latency, and HTTP fault injection.
- URL-based scenario startup through `?mockScenario=<scenario>`.
- Session-scoped persistence for scenario state and developer controls.
- Strict handling for unimplemented `/api` requests so fixture gaps fail visibly.
- Playwright browser smoke tests and MSW contract tests.

## Scenarios

The initial scenario matrix is:

| Scenario | Purpose |
| --- | --- |
| `auth-local` | Signed-out local authentication flow |
| `auth-oidc` | Signed-out OIDC provider presentation |
| `pcie-full` | Fully configured PCIe device and default development state |
| `lite-degraded` | Limited older hardware with unavailable optional services |
| `settings-lifecycle` | Preconfigured state for testing mutable settings |
| `picoclaw-lifecycle` | Uninstalled and unconfigured PicoClaw onboarding state |

The default scenario is `pcie-full` so `pnpm mocked` opens a useful authenticated desktop immediately.

## Architecture

### Fixture State

- Define serializable fixture contracts in `web/src/mocks/types.ts`.
- Keep immutable scenario factories in `web/src/mocks/scenarios.ts`.
- Keep the active observable state, persistence, reset, and faults in `web/src/mocks/state.ts`.
- Persist state in `sessionStorage` to isolate browser sessions and make reset predictable.

### Transports

- Define all REST and upload behavior in `web/src/mocks/handlers.ts`.
- Define browser WebSocket behavior in `web/src/mocks/websockets.ts`.
- Assemble and start MSW in `web/src/mocks/browser.ts` before React renders.
- Reject unexpected `/api` requests with a clear response or startup warning rather than allowing access to real hardware.

### UI Integration

- Render React exactly once after MSW is ready.
- Render the mock controls globally so they are available on protected and authentication routes.
- Replace media transport components with a static development screen only in mocked mode.
- Do not change production-mode media behavior.

## Delivery Phases

1. Build typed fixture state, scenarios, persistence, fault injection, and safe application bootstrap.
2. Implement all REST, upload, static asset, and WebSocket handlers with stateful mutations.
3. Add the static desktop and mocked-mode control drawer.
4. Add MSW contract tests for response shapes, mutations, reset, latency, and faults.
5. Add Playwright smoke tests for local auth, OIDC presentation, desktop controls, terminal, settings, and PicoClaw.
6. Document developer commands and scenario usage in the Web README.
7. Run TypeScript, lint, unit, browser, and production build verification.

## Non-Goals

- Emulating a real WebRTC peer connection or video stream.
- Emulating Direct H.264 decoding in `direct.worker.ts`.
- Bit-accurate HID or serial device emulation.
- Reproducing server timing and hardware failures beyond configurable delays and HTTP/WebSocket fixture behavior.
