package rates

import (
    "io"
    "os"
    "path/filepath"
)

func writeFileAtomically(path string, r io.Reader) error {
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
    if err != nil {
        return err
    }
    defer os.Remove(tmp.Name())
    if _, err := io.Copy(tmp, r); err != nil {
        tmp.Close()
        return err
    }
    if err := tmp.Sync(); err != nil {
        tmp.Close()
        return err
    }
    if err := tmp.Close(); err != nil {
        return err
    }
    return os.Rename(tmp.Name(), path)
}
