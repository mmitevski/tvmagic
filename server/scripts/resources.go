package main

import (
	"io/ioutil"
	"os"
	"strings"
	"log"
	"flag"
	"mime"
	"text/template"
	"path/filepath"
)

type resource struct {
	ContentType string
	Size        int
	Content     []byte
}

var files map[string]*resource
var templates map[string]*[]byte

func init() {
	files = make(map[string]*resource)
	templates = make(map[string]*[]byte)
}

func addFiles(dir string, base string) {
	fs, _ := ioutil.ReadDir(dir + base)
	for _, f := range fs {
		if f.IsDir() {
			addFiles(dir + base, base + f.Name() + "/")
			continue
		}
		name := base + f.Name()
		ext := ""
		if chunks := strings.Split(name, "."); len(chunks) > 0 {
			ext = "." + chunks[len(chunks) - 1]
		}
		mimeType := mime.TypeByExtension(ext)
		bytes, _ := ioutil.ReadFile(dir + name)
		size := len(bytes)
		files[name] = &resource{
			ContentType: mimeType,
			Content: bytes,
			Size:size,
		}
		log.Printf("Mapped file: %s\t%d\n", name, size)
		continue
	}
}

func addTemplates(dir string, base string) {
	fs, _ := ioutil.ReadDir(dir + base)
	for _, f := range fs {
		if f.IsDir() {
			addTemplates(dir + base, base + f.Name() + "/")
			continue
		}
		name := dir + base + f.Name()
		bytes, _ := ioutil.ReadFile(name)
		templates[name] = &bytes
		log.Println("Mapped template: " + name)
	}
}

type loader func(dir string)

func loadDir(locations string, loader loader) {
	if len(locations) > 0 {
		log.Printf("Processing files in [%s]...", locations)
		if dirs := strings.Split(locations, ","); len(dirs) > 0 {
			for _, dir := range dirs {
				log.Printf("Loading %s...", dir)
				loader(dir)
			}
		}
	}
}

func generate(sourceFile, targetFile string) {
	file := filepath.Base(sourceFile)
	log.Printf("Loading source template '%s'...", file)
	content, _ := ioutil.ReadFile(file)
	log.Printf("Content of the source template: %s", content)
	tpl := template.New(file).Delims("<?", "?>")
	if t, err := tpl.ParseFiles(file); err == nil {
		var data struct {
			Templates *map[string]*[]byte
			Files     *map[string]*resource
		}
		data.Templates = &templates
		data.Files = &files
		if f, err := os.Create(targetFile); err == nil {
			defer f.Close()
			err := t.Execute(f, data)
			if err != nil {
				log.Printf("Error generating target file from template %s: %s", sourceFile, err)
			}
		} else {
			log.Printf("Error creating file %s: %s", targetFile, err)
		}
	} else {
		log.Printf("Error parsing file %s: %s", sourceFile, err)
	}
}

func main() {
	templateDirs := ""
	fileDirs := ""
	sourceFile := ""
	targetFile := ""
	flag.StringVar(&templateDirs, "templates", "", "Comma-separated directories with templates")
	flag.StringVar(&fileDirs, "files", "", "Comma-separated directories with static-files")
	flag.StringVar(&sourceFile, "source", "", "Template file, used to generate the output")
	flag.StringVar(&targetFile, "target", "", "Output file")
	flag.Parse()
	log.Println("Mapping templates...")
	loadDir(templateDirs, func(dir string) {
		addTemplates(dir, "/")
	})
	loadDir(fileDirs, func(dir string) {
		addFiles(dir, "/")
	})
	generate(sourceFile, targetFile)
}