package main

import (
	"strings"
	"testing"
)

func TestReadVersion(t *testing.T) {
	embedded := strings.TrimPrefix(embeddedVersion, "v")
	tests := []struct {
		name    string
		env     string
		want    string
		wantErr bool
	}{
		{"embedded default", "", embedded, false},
		{"plain semver", "2.0.0", "2.0.0", false},
		{"v-prefix", "v1.2.3", "1.2.3", false},
		{"leading/trailing spaces", " 1.0.0 ", "1.0.0", false},
		{"v-prefix with spaces", " v1.0.0 ", "1.0.0", false},
		{"invalid: word", "latest", "", true},
		{"invalid: two parts", "1.2", "", true},
		{"invalid: four parts", "1.2.3.4", "", true},
		{"invalid: empty after trim", "   ", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CA_FFI_VERSION", tc.env)
			got, err := readVersion()
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTargetTriple(t *testing.T) {
	tests := []struct {
		goos, goarch string
		isMusl       bool
		wantTriple   string
		wantExt      string
		wantLib      string
		wantErr      bool
	}{
		{"darwin", "arm64", false, "aarch64-apple-darwin", "tar.gz", "libaptos_confidential_asset_ffi.a", false},
		{"darwin", "amd64", false, "", "", "", true},
		{"darwin", "386", false, "", "", "", true},
		{"linux", "amd64", false, "x86_64-unknown-linux-gnu", "tar.gz", "libaptos_confidential_asset_ffi.a", false},
		{"linux", "amd64", true, "x86_64-unknown-linux-musl", "tar.gz", "libaptos_confidential_asset_ffi.a", false},
		{"linux", "arm64", false, "aarch64-unknown-linux-gnu", "tar.gz", "libaptos_confidential_asset_ffi.a", false},
		{"linux", "arm64", true, "aarch64-unknown-linux-musl", "tar.gz", "libaptos_confidential_asset_ffi.a", false},
		{"linux", "386", false, "", "", "", true},
		{"windows", "amd64", false, "x86_64-pc-windows-msvc", "zip", "aptos_confidential_asset_ffi.lib", false},
		{"windows", "arm64", false, "", "", "", true},
		{"windows", "386", false, "", "", "", true},
		{"plan9", "amd64", false, "", "", "", true},
	}
	for _, tc := range tests {
		name := tc.goos + "/" + tc.goarch
		if tc.isMusl {
			name += "/musl"
		}
		t.Run(name, func(t *testing.T) {
			triple, ext, lib, err := targetTriple(tc.goos, tc.goarch, tc.isMusl)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got triple=%q", triple)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if triple != tc.wantTriple {
				t.Errorf("triple: got %q, want %q", triple, tc.wantTriple)
			}
			if ext != tc.wantExt {
				t.Errorf("ext: got %q, want %q", ext, tc.wantExt)
			}
			if lib != tc.wantLib {
				t.Errorf("lib: got %q, want %q", lib, tc.wantLib)
			}
		})
	}
}

func TestParseSHA256Sums(t *testing.T) {
	sumsData := []byte("" +
		"abc123  file-a.tar.gz\n" +
		"def456  file-b.zip\n" +
		"  \n" +
		"ghi789  file-c.tar.gz\n",
	)
	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  bool
	}{
		{"first entry", "file-a.tar.gz", "abc123", false},
		{"second entry", "file-b.zip", "def456", false},
		{"third entry", "file-c.tar.gz", "ghi789", false},
		{"missing", "file-x.tar.gz", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSHA256Sums(sumsData, tc.filename)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
