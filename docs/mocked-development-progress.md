# Mocked Web Development Progress

Last updated: 2026-08-08

## Current Status

The core mocked-mode runtime is implemented. The remaining work is automated coverage, developer workflow documentation, and final full-suite verification.

## Completed

- Audited the frontend API modules, startup authentication requests, direct resource URLs, routes, and WebSocket clients.
- Fixed mocked bootstrap so MSW starts before React and the application renders exactly once.
- Added typed fixture state with session persistence and observable updates.
- Added URL scenario selection through `mockScenario` in either the page query or hash query.
- Added six scenario factories: `auth-local`, `auth-oidc`, `pcie-full`, `lite-degraded`, `settings-lifecycle`, and `picoclaw-lifecycle`.
- Added scenario reset, global response delay, endpoint delay, persistent faults, and one-shot faults.
- Implemented stateful REST handlers for all API modules under `web/src/api`.
- Implemented upload behavior for scripts, virtual media, and offline application updates.
- Added a strict API catch-all that returns HTTP 501 for fixture gaps.
- Added browser WebSocket handlers for HID/status traffic, terminal shell/echo behavior, and PicoClaw chat behavior.
- Added a static mocked desktop that bypasses WebRTC and Direct H.264 only in mocked mode.
- Added a global mock control drawer for scenario, latency, fault, and reset controls.

## Verification Completed

- `npm exec tsc -- --noEmit`
- Targeted ESLint for `src/mocks`, `src/main.tsx`, and the mocked screen integration
- A mocked Vite production build was completed after the transport implementation; another final build is required after all UI and test changes.

`pnpm` is not available directly in the current shell, so the completed local checks used the installed npm tooling.

## Remaining

- Add MSW contract tests for every endpoint group and important state transition.
- Add Playwright and browser smoke tests.
- Cover local authentication and OIDC login presentation.
- Cover the default desktop, mock controls, terminal, settings lifecycle, and PicoClaw lifecycle.
- Add mocked-mode usage, scenario URLs, and fault injection instructions to `web/README.md`.
- Add or update CI commands for Playwright where appropriate.
- Run the complete lint, unit test, Playwright, and build suite.
- Resolve any response-shape issues discovered by browser smoke tests.

## Known Boundaries

- Mocked mode intentionally uses a static desktop instead of real WebRTC, Direct H.264, or hardware MJPEG.
- The terminal fixture provides a deterministic shell banner, input echo, and prompt; it is not a shell emulator.
- PicoClaw returns deterministic assistant messages and records them in fixture history; it does not invoke an AI model.
- Browser interactions remain the source of truth for response-shape compatibility because many existing API wrappers are not strongly typed.

## Resume Point

Start with MSW contract tests and Playwright setup. Use `pcie-full` for the primary desktop smoke path, then exercise `auth-local`, `auth-oidc`, `settings-lifecycle`, and `picoclaw-lifecycle` through `?mockScenario=<name>`.
