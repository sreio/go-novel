package epub

import (
    "fmt"
    e "github.com/bmaupin/go-epub"
)

type Meta struct { Title, Author string }
type Chapter struct { Title, Content string }

func Save(path string, meta Meta, chapters []Chapter) error {
    book := e.NewEpub(meta.Title)
    if meta.Author != "" { book.SetAuthor(meta.Author) }
    for _, ch := range chapters {
        // 简单章节包装；可根据需要加样式
        _, err := book.AddSection(fmt.Sprintf("<h1>%s</h1><p>%s</p>", ch.Title, ch.Content), ch.Title, "", "")
        if err != nil { return err }
    }
    return book.Write(path)
}
