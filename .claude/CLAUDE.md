# Codebase Guide for AI Agents

## Repo structure

This is a monorepo with multiple language bindings over a Rust FFI core.

```
rust/ffi/          — Rust FFI library (cbindgen → include/aptos_confidential_asset.h)
bindings/go/       — Go module (CGO, independent go.mod)
bindings/python/   — Python wheel (PyO3/maturin)
bindings/js/       — JS/WASM bindings
examples/go/       — Runnable Go example
```

## Release process

### FFI release (prebuilt .a / .lib artifacts)

Pushing a `v*.*.*` tag triggers `bindings-release.yml`, which builds static
libraries for all supported platforms and uploads them to GitHub Releases
along with a `SHA256SUMS` file.

### Go module release

The Go module lives in a subdirectory (`bindings/go/go.mod`), so it needs its
own tag with a path prefix — otherwise `go get` cannot find the version:

```bash
git tag bindings/go/v1.1.2 <merge-commit>
git push upstream bindings/go/v1.1.2
```

The version number matches the FFI release. `bindings/go/VERSION` stores the
current version (no `v` prefix).

### Python wheel release

The same `v*.*.*` tag also triggers `publish-python.yml`, building
manylinux_2_28 + macOS universal2 + Windows wheels and publishing to PyPI via
OIDC Trusted Publisher.

### Typical release checklist

1. Merge PR into `upstream/main`
2. Push `v1.1.2` → triggers FFI build + Python wheel publish
3. Push `bindings/go/v1.1.2` → enables `go get .../bindings/go@v1.1.2`

## Go bindings — how CGO finds the native library

CGO files use:
- `CFLAGS: -I${SRCDIR}/include` — header always present in the repo
- `LDFLAGS: -laptos_confidential_asset_ffi` — no `-L` path; caller must set `CGO_LDFLAGS`

**In-repo development** — stage the built `.a` manually:

```bash
cargo build -p aptos_confidential_asset_ffi --release --manifest-path rust/Cargo.toml
TRIPLE=aarch64-apple-darwin
mkdir -p bindings/go/aptosconfidential/native/$TRIPLE
cp rust/target/release/libaptos_confidential_asset_ffi.a bindings/go/aptosconfidential/native/$TRIPLE/
cd bindings/go && CGO_LDFLAGS="-L$(pwd)/aptosconfidential/native/$TRIPLE" go test ./aptosconfidential/...
```

**External consumer** — download prebuilt library, then build:

```bash
go run github.com/aptos-labs/confidential-asset-bindings/bindings/go/aptosconfidential/tools/download@v1.1.2
CGO_LDFLAGS="-L$(pwd)/native/aarch64-apple-darwin" go build ./...
```

## FFI header sync rule

`rust/ffi/include/aptos_confidential_asset.h` is the source of truth (generated
by cbindgen). `bindings/go/aptosconfidential/include/aptos_confidential_asset.h`
is a committed copy so external `go get` users always have the header available.

**When changing the Rust FFI interface, always run:**

```bash
cp rust/ffi/include/aptos_confidential_asset.h \
   bindings/go/aptosconfidential/include/aptos_confidential_asset.h
```

CI (`check-ffi-header-sync` job) will fail if the two files diverge.

## Fork vs upstream

- upstream: `aptos-labs/confidential-asset-bindings`
- Feature branches must be based on `upstream/main` (not the fork's main — they have diverged history)
