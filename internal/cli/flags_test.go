package cli

import (
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "testing"
)

func TestResolveConfigFilePath_BareNameVariants(t *testing.T) {
    base := t.TempDir()
    home := filepath.Join(base, "home")
    configsDir := filepath.Join(home, ".config", "fabric", "configs")
    if err := os.MkdirAll(configsDir, 0o755); err != nil {
        t.Fatalf("failed to create configs dir: %v", err)
    }
    // Create candidates
    if err := os.WriteFile(filepath.Join(configsDir, "raw"), []byte("k: v\n"), 0o644); err != nil {
        t.Fatalf("failed to write raw: %v", err)
    }
    if err := os.WriteFile(filepath.Join(configsDir, "dev.yaml"), []byte("k: v\n"), 0o644); err != nil {
        t.Fatalf("failed to write dev.yaml: %v", err)
    }
    if err := os.WriteFile(filepath.Join(configsDir, "prod.yml"), []byte("k: v\n"), 0o644); err != nil {
        t.Fatalf("failed to write prod.yml: %v", err)
    }

    t.Setenv("HOME", home)
    // On Windows, some Go versions prefer USERPROFILE
    if runtime.GOOS == "windows" {
        t.Setenv("USERPROFILE", home)
    }

    tests := []struct {
        in       string
        wantBase string // base filename expected to resolve to
    }{
        {"raw", "raw"},
        {"dev", "dev.yaml"},
        {"prod", "prod.yml"},
    }

    for _, tc := range tests {
        got, err := resolveConfigFilePath(tc.in)
        if err != nil {
            t.Fatalf("resolveConfigFilePath(%q) unexpected error: %v", tc.in, err)
        }
        if !strings.HasSuffix(got, filepath.Join("configs", tc.wantBase)) {
            t.Fatalf("resolveConfigFilePath(%q) = %q; want path ending with %q", tc.in, got, filepath.Join("configs", tc.wantBase))
        }
    }

    if _, err := resolveConfigFilePath("missing"); err == nil {
        t.Fatalf("resolveConfigFilePath(%q) expected error, got nil", "missing")
    }
}

func TestResolveConfigFilePath_AbsoluteAndRelative(t *testing.T) {
    base := t.TempDir()
    home := filepath.Join(base, "home")
    t.Setenv("HOME", home)
    if runtime.GOOS == "windows" {
        t.Setenv("USERPROFILE", home)
    }

    // Absolute path
    absFile := filepath.Join(base, "abs.yaml")
    if err := os.WriteFile(absFile, []byte("x: 1\n"), 0o644); err != nil {
        t.Fatalf("write abs file: %v", err)
    }
    got, err := resolveConfigFilePath(absFile)
    if err != nil || got != absFile {
        t.Fatalf("resolve absolute: got=%q err=%v; want %q", got, err, absFile)
    }

    // Relative path with separator should pass through unchanged
    relDir := filepath.Join(base, "rel")
    if err := os.MkdirAll(relDir, 0o755); err != nil {
        t.Fatalf("mkdir rel dir: %v", err)
    }
    relFile := filepath.Join(".", filepath.Base(relDir), "c.yaml")
    // Create actual file at the absolute location
    if err := os.WriteFile(filepath.Join(relDir, "c.yaml"), []byte("a: b\n"), 0o644); err != nil {
        t.Fatalf("write rel file: %v", err)
    }
    got, err = resolveConfigFilePath(relFile)
    if err != nil || got != relFile {
        t.Fatalf("resolve relative: got=%q err=%v; want %q", got, err, relFile)
    }
}

func TestReadSchemaFile_SearchInSchemasDir(t *testing.T) {
    base := t.TempDir()
    home := filepath.Join(base, "home")
    schemasDir := filepath.Join(home, ".config", "fabric", "schemas")
    if err := os.MkdirAll(schemasDir, 0o755); err != nil {
        t.Fatalf("failed to create schemas dir: %v", err)
    }
    // Write both without and with .json
    if err := os.WriteFile(filepath.Join(schemasDir, "alpha.json"), []byte("{\"a\":1}"), 0o644); err != nil {
        t.Fatalf("write alpha.json: %v", err)
    }
    if err := os.WriteFile(filepath.Join(schemasDir, "beta"), []byte("{\"b\":2}"), 0o644); err != nil {
        t.Fatalf("write beta: %v", err)
    }

    t.Setenv("HOME", home)
    if runtime.GOOS == "windows" {
        t.Setenv("USERPROFILE", home)
    }

    // Should find exact file in schemas dir
    s, err := readSchemaFile("alpha.json")
    if err != nil || strings.TrimSpace(s) != "{\"a\":1}" {
        t.Fatalf("readSchemaFile alpha.json: got=(%q,%v); want %q,nil", s, err, "{\"a\":1}")
    }

    // Should append .json if missing
    s, err = readSchemaFile("alpha")
    if err != nil || strings.TrimSpace(s) != "{\"a\":1}" {
        t.Fatalf("readSchemaFile alpha: got=(%q,%v); want %q,nil", s, err, "{\"a\":1}")
    }

    // Should read file without extension as-is
    s, err = readSchemaFile("beta")
    if err != nil || strings.TrimSpace(s) != "{\"b\":2}" {
        t.Fatalf("readSchemaFile beta: got=(%q,%v); want %q,nil", s, err, "{\"b\":2}")
    }
}

func TestReadSchemaFile_AbsolutePath(t *testing.T) {
    base := t.TempDir()
    f := filepath.Join(base, "schema.json")
    want := "{\"k\":\"v\"}"
    if err := os.WriteFile(f, []byte(want), 0o644); err != nil {
        t.Fatalf("write schema: %v", err)
    }

    got, err := readSchemaFile(f)
    if err != nil {
        t.Fatalf("readSchemaFile absolute returned error: %v", err)
    }
    if strings.TrimSpace(got) != want {
        t.Fatalf("readSchemaFile absolute: got %q want %q", got, want)
    }
}

