package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"golang.org/x/net/html"
)

type Doc struct {
	Root *html.Node
}

func ParseDoc(htmlStr string) *Doc {
	htmlStr = strings.TrimSpace(htmlStr)
	if htmlStr == "" {
		return nil
	}
	node, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing HTML: %v", err)
		return nil
	}
	return &Doc{Root: node}
}

func Docf(format string, args ...interface{}) *Doc {
	return ParseDoc(fmt.Sprintf(format, args...))
}

func (d *Doc) GoDoc() string {
	var buf strings.Builder
	walkComment(&buf, d.Root)
	return buf.String()
}

func PrintComment(w io.Writer, comment string) {
	width := 72

	// Ensure no indented block breaks
	for _, l := range strings.Split(comment, "\n") {
		if strings.HasPrefix(l, "  ") {
			if n := len(l); n > width {
				width = n
			}
		}
	}

	comment = wordwrap.WrapString(comment, uint(width))
	lines := strings.Split(comment, "\n")
	for i, line := range lines {
		lines[i] = "// " + line
	}
	fmt.Fprintln(w, strings.Join(lines, "\n"))
}

func walkComment(w io.Writer, node *html.Node) {
	var n int
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			if c.Data == " " {
				continue
			}
			fmt.Fprint(w, c.Data)
		case html.ElementNode:
			switch strings.ToLower(c.Data) {
			case "a":
				walkComment(w, c)
				if afterSee(c) {
					fmt.Fprint(w, ": "+attr(c, "href"))
				}
			case "p":
				br(w, c)
				walkComment(w, c)
			case "b", "strong":
				if isChildOf(c, "p") && attr(c.Parent, "class") == "title" {
					fmt.Fprint(w, "\n\n")
					walkComment(w, c)
					fmt.Fprint(w, "\n\n")
					continue
				}
				walkComment(w, c)
			case "h1", "h2", "h3", "h4", "h5", "h6":
				br(w, c)
				walkComment(w, c)
				fmt.Fprint(w, "\n\n")
			case "hr":
				br(w, c)
				fmt.Fprint(w, "\n---\n\n")
			case "ul", "ol":
				br(w, c)
				walkComment(w, c)
				fmt.Fprint(w, "\n")
			case "li":
				br(w, c)
				if isChildOf(c, "ul") {
					fmt.Fprint(w, "  ")
				} else if isChildOf(c, "ol") {
					n++
					fmt.Fprint(w, fmt.Sprintf("  %d. ", n))
				}
				// var li strings.Builder
				walkComment(w, c)
				fmt.Fprint(w, "\n")
				// fmt.Fprintln(w, strings.TrimSpace(li.String()))
			default:
				walkComment(w, c)
			}
		}
	}
}

func attr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func afterSee(node *html.Node) bool {
	node = node.PrevSibling
	if node == nil {
		return false
	}
	if node.Type == html.TextNode {
		str := strings.TrimSpace(node.Data)
		return strings.HasSuffix(str, "see")
	}
	return false
}

func br(w io.Writer, node *html.Node) {
	node = node.PrevSibling
	if node == nil {
		return
	}
	switch node.Type {
	case html.TextNode:
		text := strings.Trim(node.Data, " \t")
		if text != "" && !strings.HasSuffix(text, "\n") {
			fmt.Fprint(w, "\n")
		}
	case html.ElementNode:
		switch strings.ToLower(node.Data) {
		case "br", "p", "ul", "ol", "div", "blockquote", "h1", "h2", "h3", "h4", "h5", "h6":
			fmt.Fprint(w, "\n")
		}
	}
}

func isChildOf(node *html.Node, parents ...string) bool {
	node = node.Parent
	if node == nil {
		return false
	}
	for _, name := range parents {
		if node.Type == html.ElementNode && strings.ToLower(node.Data) == name {
			return true
		}
	}
	return false
}
