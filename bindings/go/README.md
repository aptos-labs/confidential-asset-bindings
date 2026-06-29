# Go bindings (`aptosconfidential`)

**Experimental.** Module path: `github.com/aptos-labs/confidential-asset-bindings/bindings/go`.

## Quick start (external project)

```bash
# 1. Add the dependency
go get github.com/aptos-labs/confidential-asset-bindings/bindings/go@v1.1.2

# 2. Download the prebuilt native library into your project directory
#    (only needed once per version)
go run github.com/aptos-labs/confidential-asset-bindings/bindings/go/aptosconfidential/tools/download@v1.1.2

# 3. Build — point the linker at the downloaded library
CGO_LDFLAGS="-L$(pwd)/native/aarch64-apple-darwin" go build ./...
```

Step 2 downloads a prebuilt static library from GitHub Releases into `native/<triple>/` in your current directory. No Rust toolchain required.

The triple in step 3 matches your platform — see the [platform table](#supported-platforms) below.

## Quick start (in this repo)

```bash
# 1. Build the Rust FFI library
cargo build -p aptos_confidential_asset_ffi --release --manifest-path rust/Cargo.toml

# 2. Stage the built library (Linux/macOS)
TRIPLE=aarch64-apple-darwin  # adjust for your platform
mkdir -p bindings/go/aptosconfidential/native/$TRIPLE
cp rust/target/release/libaptos_confidential_asset_ffi.a bindings/go/aptosconfidential/native/$TRIPLE/
# Windows: cp rust\target\release\aptos_confidential_asset_ffi.lib bindings\go\aptosconfidential\native\$TRIPLE\

# 3. Test
cd bindings/go && go test ./aptosconfidential/...
```

For musl Linux, use `-tags musl` and set `TRIPLE=x86_64-unknown-linux-musl` (or `aarch64-unknown-linux-musl`). When running the download tool on a glibc host targeting musl, set `CA_MUSL=1` so it downloads the musl variant (auto-detection only works on musl hosts).

## Usage

```go
import "github.com/aptos-labs/confidential-asset-bindings/bindings/go/aptosconfidential"

blindingsFlat, err := aptosconfidential.FlattenBlindings([][]byte{r0, r1})
if err != nil { /* ... */ }
proof, commsFlat, err := aptosconfidential.BatchRangeProof(values, blindingsFlat, valBase32, randBase32, 32)
ok, err := aptosconfidential.BatchVerifyProof(proof, commsFlat, valBase32, randBase32, 32)

solver := aptosconfidential.NewSolver()
value, err := solver.Solve(compressedPoint32, 32)
```

`numBits` for `BatchRangeProof`/`BatchVerifyProof` must be one of `8, 16, 32, 64`. `Solver.Solve` accepts `maxNumBits` of `16` or `32` only, and returns an error if the value is invalid or the solver is nil or already closed.

For long-running services, call `(*Solver).Close()` explicitly to release native resources deterministically.

See [examples/go](../../examples/go) for runnable examples.

## Supported platforms

| Platform | Triple | Prebuilt download | Notes |
|---|---|---|---|
| macOS arm64 (M1/M2/M3) | `aarch64-apple-darwin` | Yes | |
| macOS amd64 (Intel) | `x86_64-apple-darwin` | No — build from source | |
| Linux amd64 (glibc) | `x86_64-unknown-linux-gnu` | Yes | |
| Linux amd64 (musl) | `x86_64-unknown-linux-musl` | Yes | `-tags musl` required |
| Linux arm64 (glibc) | `aarch64-unknown-linux-gnu` | Yes | |
| Linux arm64 (musl) | `aarch64-unknown-linux-musl` | Yes | `-tags musl` required |
| Windows amd64 | `x86_64-pc-windows-msvc` | Yes | |
| Windows arm64 | `aarch64-pc-windows-msvc` | No — build from source | |

## Building from source (unsupported platforms)

For platforms marked "No — build from source" (Intel macOS, Windows arm64):

```bash
# 1. Clone the repo and build the Rust FFI library
git clone https://github.com/aptos-labs/confidential-asset-bindings
cargo build -p aptos_confidential_asset_ffi --release \
  --manifest-path confidential-asset-bindings/rust/Cargo.toml \
  --target x86_64-apple-darwin  # replace with your triple

# 2. Copy the library into your project
mkdir -p ./native/x86_64-apple-darwin
cp confidential-asset-bindings/rust/target/x86_64-apple-darwin/release/libaptos_confidential_asset_ffi.a \
   ./native/x86_64-apple-darwin/

# 3. Build your project, pointing the linker at the library
CGO_LDFLAGS="-L$(pwd)/native/x86_64-apple-darwin" go build ./...
```

The Go module already contains the header file — no separate header copy is needed.

## Prerequisites

- `CGO_ENABLED=1` (default on native builds)
- A C compiler (Clang on macOS, GCC on Linux, MinGW-w64/GCC on Windows — CGO requires a GCC-compatible toolchain; MinGW-w64 can link the MSVC-format `.lib` prebuilts)

## Version pinning

Override the downloaded native library version with the `CA_FFI_VERSION` environment variable:

```bash
CA_FFI_VERSION=1.1.1 go run github.com/aptos-labs/confidential-asset-bindings/bindings/go/aptosconfidential/tools/download@v1.1.2
```

> **Warning:** The Go module ships a header file matching its own release. Overriding `CA_FFI_VERSION` downloads a different library version but keeps the module's original header. Use a matching `@vX.Y.Z` module version to avoid ABI mismatches.
