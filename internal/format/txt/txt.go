package txt

import (
    "os"
)

type Chapter struct { Title, Content string }

func Save(path string, chapters []Chapter) error {
    f, err := os.Create(path)
    if err != nil { return err }
    defer f.Close()
    for _, ch := range chapters {
        if _, err := f.WriteString(ch.Title + "\n\n" + ch.Content + "\n\n"); err != nil { return err }
    }
    return nil
}
