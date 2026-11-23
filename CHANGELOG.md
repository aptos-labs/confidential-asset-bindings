# @aptos-labs/confidential-asset-bindings

## 1.1.3

### Patch Changes

- 289062d: Update dependencies to resolve known security vulnerabilities (CVEs).

## 1.1.2

### Patch Changes

- bbbfee6: Consolidate native bindings into `aptos_confidential_asset_ffi` (iOS/Android rebuild; Android loads `libaptos_confidential_asset_ffi`).
- bbbfee6: Harden CI and release: require changesets for `rust/ffi`, add Android JNI compile smoke, publish `aarch64-unknown-linux-musl` FFI artifacts, verify FFI git tags match `package.json`, serialize native solver calls with a mutex, and unify npm publish with automatic `v*` tag + native FFI GitHub Release (`release.yml` → `bindings-release.yml`).

## 1.1.1

### Patch Changes

- 8f50786: fix: resolve node:fs bundler errors in browser environments by providing a dedicated browser build via the `browser` export condition

  Bundlers (webpack, Vite, Turbopack) statically analyse dynamic imports regardless of runtime guards. The `node:fs` import in the shared WASM loader was being pulled into browser bundles, causing build-time or runtime errors.

  The fix introduces a proper source split:

  - `src/web/shared.ts` — WASM API factory with no environment-specific code
  - `src/web/index.node.ts` — Node.js entry: resolves WASM from local `node_modules` via `node:fs`, falls back to CDN
  - `src/web/index.browser.ts` — Browser entry: unconditionally uses CDN URL; zero `node:fs` references

  The `browser` export condition in `package.json` routes all bundler targets to `index.browser.mjs`, so no bundler-specific magic comments are needed.

## 1.0.0

### Major Changes

- a258d0f: Add React Native / Expo native module with full iOS and Android support.

  The package now runs natively on iOS (C FFI → xcframework) and Android (JNI → .so) via Expo's JSI layer, with no WASM involved on mobile. The public API (`solveDiscreteLog`, `batchRangeProof`, `batchVerifyProof`) is identical across all platforms. Web and Node.js continue to use the existing WASM path.

## 0.1.0

### Minor Changes

- 63334cd: Initial release of `@aptos-labs/confidential-asset-bindings`.

  This package provides TypeScript bindings for Aptos confidential asset cryptography.

  ### Discrete Log

  - `solveDiscreteLog(y, maxNumBits)` — solves discrete logarithms on the Ristretto255 curve for 16-bit and 32-bit secrets;
  - Default algorithm: TBSGS-k32 (~774 KiB WASM) — configurable at compile time via Cargo features (`tbsgs_k`, `bsgs_k`, `bsgs`, `bl12`)

  ### Range Proofs (Bulletproofs)

  - `rangeProof(inputs)` / `verifyProof(inputs)` — single range proof generation and verification
  - `batchRangeProof(inputs)` / `batchVerifyProof(inputs)` — batched variants
  - Supports 8, 16, 32, and 64-bit ranges using Pedersen commitments with custom base points
  - Proofs are cross-version compatible with bulletproofs v4.0.0 (as used in aptos-core)

  ### Architecture

  - Pure Rust core (`aptos_confidential_asset_core`) separated from the WASM runtime crate (`aptos_confidential_asset_wasm`) for reusability
  - Ships ESM and CJS bundles with full TypeScript type definitions
