// Download prebuilt native FFI static libraries from GitHub Releases.
// Run via: go generate ./aptosconfidential/... (from bindings/go/)
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	releaseBaseURL = "https://github.com/aptos-labs/confidential-asset-bindings/releases/download"
	nativeDir      = "./native" // relative to CWD (project root for external consumers, aptosconfidential/ via go generate)
	httpTimeout    = 120 * time.Second
)

var httpClient = &http.Client{Timeout: httpTimeout}

func main() {
	version, err := readVersion()
	if err != nil {
		fatalf("%v", err)
	}

	goos := envOr("GOOS", runtime.GOOS)
	goarch := envOr("GOARCH", runtime.GOARCH)
	isMusl := detectMusl(goos)

	triple, ext, libName, err := targetTriple(goos, goarch, isMusl)
	if err != nil {
		if goos == "windows" {
			fatalf("%v\n\nTo build from source (replace <triple> with your target, e.g. aarch64-pc-windows-msvc):\n"+
				"  git clone https://github.com/aptos-labs/confidential-asset-bindings\n"+
				"  cargo build -p aptos_confidential_asset_ffi --release"+
				" --manifest-path confidential-asset-bindings\\rust\\Cargo.toml --target <triple>\n"+
				"  mkdir native\\<triple>\n"+
				"  copy confidential-asset-bindings\\rust\\target\\<triple>\\release\\aptos_confidential_asset_ffi.lib native\\<triple>\\\n"+
				"  set CGO_LDFLAGS=-L%%cd%%\\native\\<triple> && go build ./...",
				err)
		} else {
			fatalf("%v\n\nTo build from source (replace <triple> with your target, e.g. x86_64-apple-darwin):\n"+
				"  git clone https://github.com/aptos-labs/confidential-asset-bindings\n"+
				"  cargo build -p aptos_confidential_asset_ffi --release \\\n"+
				"    --manifest-path confidential-asset-bindings/rust/Cargo.toml --target <triple>\n"+
				"  mkdir -p ./native/<triple>\n"+
				"  cp confidential-asset-bindings/rust/target/<triple>/release/lib*.a ./native/<triple>/\n"+
				"  CGO_LDFLAGS=\"-L$(pwd)/native/<triple>\" go build ./...",
				err)
		}
	}

	outDir := filepath.Join(nativeDir, triple)
	sentinel := filepath.Join(outDir, ".version")

	if existing, err2 := os.ReadFile(sentinel); err2 == nil {
		if strings.TrimSpace(string(existing)) == version {
			fmt.Printf("native/%s already at v%s, skipping\n", triple, version)
			return
		}
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatalf("mkdir %s: %v", outDir, err)
	}

	archiveName := fmt.Sprintf("aptos_confidential_asset_ffi-%s.%s", triple, ext)
	tagURL := fmt.Sprintf("%s/v%s", releaseBaseURL, version)

	fmt.Printf("Downloading native/%s for v%s...\n", triple, version)

	sumsData, err := httpGet(tagURL + "/SHA256SUMS")
	if err != nil {
		fatalf("download SHA256SUMS: %v", err)
	}
	expectedHash, err := parseSHA256Sums(sumsData, archiveName)
	if err != nil {
		fatalf("%v", err)
	}

	archiveData, err := httpGet(tagURL + "/" + archiveName)
	if err != nil {
		fatalf("download %s: %v", archiveName, err)
	}

	actualHash := fmt.Sprintf("%x", sha256.Sum256(archiveData))
	if actualHash != expectedHash {
		fatalf("SHA256 mismatch for %s:\n  expected: %s\n  actual:   %s",
			archiveName, expectedHash, actualHash)
	}

	if ext == "zip" {
		err = extractZip(archiveData, outDir, triple, libName)
	} else {
		err = extractTarGz(archiveData, outDir, triple, libName)
	}
	if err != nil {
		fatalf("extract: %v", err)
	}

	// Verify expected outputs are present before writing the sentinel.
	for _, name := range []string{libName, "aptos_confidential_asset.h"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			fatalf("extraction succeeded but %s is missing in %s", name, outDir)
		}
	}

	if err := os.WriteFile(sentinel, []byte(version+"\n"), 0o644); err != nil {
		fatalf("write sentinel: %v", err)
	}
	fmt.Printf("native/%s: ready (v%s)\n", triple, version)
}

var semverRE = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func readVersion() (string, error) {
	raw := embeddedVersion
	if v := os.Getenv("CA_FFI_VERSION"); v != "" {
		raw = v
	}
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "v"))
	if !semverRE.MatchString(raw) {
		return "", fmt.Errorf("invalid version %q: must be X.Y.Z", raw)
	}
	return raw, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func detectMusl(goos string) bool {
	if os.Getenv("CA_MUSL") == "1" {
		return true
	}
	if goos != "linux" {
		return false
	}
	if strings.Contains(os.Getenv("GOFLAGS"), "musl") {
		return true
	}
	matches, _ := filepath.Glob("/lib/ld-musl-*")
	return len(matches) > 0
}

func targetTriple(goos, goarch string, isMusl bool) (triple, ext, libName string, err error) {
	switch goos {
	case "darwin":
		ext, libName = "tar.gz", "libaptos_confidential_asset_ffi.a"
		switch goarch {
		case "arm64":
			triple = "aarch64-apple-darwin"
		case "amd64":
			err = fmt.Errorf("no prebuilt for x86_64-apple-darwin (Intel macOS); build from source")
		default:
			err = fmt.Errorf("unsupported darwin arch: %s", goarch)
		}
	case "linux":
		ext, libName = "tar.gz", "libaptos_confidential_asset_ffi.a"
		switch goarch {
		case "amd64":
			if isMusl {
				triple = "x86_64-unknown-linux-musl"
			} else {
				triple = "x86_64-unknown-linux-gnu"
			}
		case "arm64":
			if isMusl {
				triple = "aarch64-unknown-linux-musl"
			} else {
				triple = "aarch64-unknown-linux-gnu"
			}
		default:
			err = fmt.Errorf("unsupported linux arch: %s", goarch)
		}
	case "windows":
		ext, libName = "zip", "aptos_confidential_asset_ffi.lib"
		switch goarch {
		case "amd64":
			triple = "x86_64-pc-windows-msvc"
		case "arm64":
			err = fmt.Errorf("no prebuilt for aarch64-pc-windows-msvc (Windows arm64); build from source")
		default:
			err = fmt.Errorf("unsupported windows arch: %s", goarch)
		}
	default:
		err = fmt.Errorf("unsupported OS: %s", goos)
	}
	return
}

func parseSHA256Sums(data []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no entry for %q in SHA256SUMS", filename)
}

func httpGet(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

func extractTarGz(data []byte, outDir, triple, libName string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	wantPrefix := triple + "/"
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !strings.HasPrefix(hdr.Name, wantPrefix) {
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base != libName && base != "aptos_confidential_asset.h" {
			continue
		}
		if err := writeFile(filepath.Join(outDir, base), tr); err != nil {
			return fmt.Errorf("write %s: %w", base, err)
		}
	}
	return nil
}

func extractZip(data []byte, outDir, triple, libName string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	wantPrefix := triple + "/"
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, wantPrefix) {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 || f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		if base != libName && base != "aptos_confidential_asset.h" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err2 := writeFile(filepath.Join(outDir, base), rc)
		rc.Close()
		if err2 != nil {
			return fmt.Errorf("write %s: %w", base, err2)
		}
	}
	return nil
}

func writeFile(dest string, r io.Reader) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".dl-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, copyErr := io.Copy(tmp, r)
	closeErr := tmp.Close()
	if copyErr != nil {
		os.Remove(tmpName)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmpName)
		return closeErr
	}
	os.Remove(dest) // pre-remove so os.Rename succeeds on Windows when dest exists
	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return err
	}
	// os.CreateTemp uses 0600; make library and header readable by all users.
	_ = os.Chmod(dest, 0644)
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
