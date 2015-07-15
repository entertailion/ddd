package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	http.HandleFunc("/dream", func(w http.ResponseWriter, r *http.Request) {
		cmd := exec.Command("ipython", "/ddd/deepdreams.py")
		cmd.Stdin = r.Body
		cmd.Stdout = w
		err := cmd.Run()
		if err != nil {
			log.Printf("error running command: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	http.HandleFunc("/images", func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean(r.URL.Path)
		if !strings.HasPrefix(p, "/images") {
			http.Error(w, "invalid path", http.StatusBadRequest)
		}
		f, err := os.Open(p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		defer f.Close()
		io.Copy(w, f)
	})
}
func main() {
	http.ListenAndServe(":8080", nil)
}
