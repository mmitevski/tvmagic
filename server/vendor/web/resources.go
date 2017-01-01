package web

import (
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"fmt"
)

//go:generate go run ../scripts/resources.go --templates layout,pages --files static --source static.go.template --target static.go

type resource struct {
	ContentType string
	Size        int
	Content     *[]byte
}

var resources struct {
	files map[string]*resource
    templates map[string]*[]byte
}

func register(alias string) {
	http.HandleFunc(alias, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "Keep-Alive")
		w.Header().Set("Keep-Alive", "timeout=5, max=100")
		http.ServeFile(w, r, "web/static"+alias)
	})
}

func listFiles(dir string, base string) {
	fs, _ := ioutil.ReadDir(dir + base)
	for _, f := range fs {
		if f.IsDir() {
			listFiles(dir+base, base+f.Name()+"/")
			continue
		}
		if strings.HasSuffix(base, "fonts/") ||
			strings.HasSuffix(f.Name(), ".js") ||
			strings.HasSuffix(f.Name(), ".css") ||
			strings.HasSuffix(f.Name(), ".png") {
			name := base + f.Name()
			log.Println("Mapped resource: " + name)
			register(name)
			continue
		}
	}
}

var lastModified = func() func() time.Time {
	var lastModified = time.Now().Round(time.Second).UTC()
	return func() time.Time {
		return lastModified
	}
}()

func executeOnRequestedHeader(header string, w http.ResponseWriter, r *http.Request, handler http.HandlerFunc) {
	if value := r.Header.Get("If-Modified-Since"); value != "" {
		handler(w, r)
	}
}

func codeResourceHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if resource, ok := resources.files[path]; ok {
		content := resource.Content
		if modifiedSince := r.Header.Get("If-Modified-Since"); modifiedSince != "" {
			t, err := time.Parse(time.RFC1123, modifiedSince)
			if err == nil && (t.Second() >= lastModified().Second()){
				w.WriteHeader(http.StatusNotModified)
				return
			} else {
				log.Printf("since '%s', actual: %s", t, lastModified())
			}
		}
		w.Header().Set("Last-Modified", lastModified().Format(time.RFC1123))
		w.Header().Set("Connection", "Keep-Alive")
		w.Header().Set("Keep-Alive", "timeout=5, max=100")
		w.Header().Set("Content-Type", resource.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(*content)))
		w.Write(*content)
		log.Printf("Served '%s' from the codebase", path)
	} else {
		log.Printf("Not found '%s' from the codebase", path)
	}
}

func Register() {
	if resources.files != nil { // files are embedded in the code
		for alias, _ := range resources.files {
			http.Handle(alias, http.HandlerFunc(codeResourceHandler))
			//http.Handle(alias, gziphandler.GzipHandler(http.HandlerFunc(codeResourceHandler)))
			log.Printf("Mapped '%s' from the codebase", alias)
		}
	} else {
		listFiles("web/static", "/")
	}
}

var templates = make(map[string]*template.Template, 0)

func Template(alias string, funcMap template.FuncMap) (*template.Template) {
	if resources.templates != nil { // resources are embedded in the code
		if tpl, ok := templates[alias]; ok {
			log.Printf("Found template for '%v'", alias)
			return tpl
		}
	}
	tpl := template.New(filepath.Base(alias)).Delims("<?", "?>")
	if funcMap != nil {
		tpl = tpl.Funcs(funcMap)
	}
	var t *template.Template
	var err error
	if resources.templates != nil { // resources are embedded in the code
		var s string
		if b, ok := resources.templates[alias]; ok {
			log.Printf("Found template for '%v' in the codebase", alias)
			s = string(*b)
		} else {
			log.Printf("Error finding resource for template: %s", err.Error())
			return nil
		}
		t, err = tpl.Parse(s)
	} else {
		t, err = tpl.ParseFiles("web/" + alias)
	}
	if err != nil {
		log.Printf("Error parsing template: %s", err.Error())
		return nil
	}
	templates[alias] = t
	return t
}
