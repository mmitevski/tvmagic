package web

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"text/template"
	"log"
	"strings"
	"services/session"
)

type MainPageData struct {
	MainPage      bool
	Heading       string
	Content       string
	Path          string
	Authenticated bool
}

func (d *MainPageData) Selected(path string) string {
	if path == "/" {
		if d.Path == path {
			return "active"
		}
	} else if strings.HasPrefix(d.Path, path) {
		return "active"
	}
	return ""
}

func MainLayout(w http.ResponseWriter, r *http.Request, heading string, handler func(io.Writer)) {
	data := &MainPageData{}
	data.MainPage = (r.URL.Path == "/")
	data.Heading = heading
	data.Path = r.URL.Path
	data.Authenticated = session.GetAuthentication(r) != nil
	var buffer bytes.Buffer
	out := bufio.NewWriter(&buffer)
	handler(out)
	out.Flush()
	data.Content = buffer.String()
	data.Path = r.RequestURI
	funcMap := template.FuncMap{
	}
	t := Template("layout/main.html", funcMap)
	if s := t.Execute(w, data); s != nil {
		log.Println(s)
	}
}

func Layout(alias string, w io.Writer, r *http.Request, data interface{}) {
	t := Template(alias, nil)
	if s := t.Execute(w, data); s != nil {
	}
}
