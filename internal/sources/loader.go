package sources

import (
    "io/fs"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

// LoadFromDir 读取目录下所有 .yaml/.yml
func LoadFromDir(dir string) ([]SourceConfig, error) {
    var out []SourceConfig
    err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
        if err != nil { return err }
        if d.IsDir() { return nil }
        ext := filepath.Ext(path)
        if ext != ".yaml" && ext != ".yml" { return nil }
        b, err := os.ReadFile(path)
        if err != nil { return err }
        var cfg SourceConfig
        if err := yaml.Unmarshal(b, &cfg); err != nil { return err }
        out = append(out, cfg)
        return nil
    })
    return out, err
}
