package main

import (
	"bufio"
	"fmt"
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
		stdout, err := cmd.StdoutPipe()
		stderr, err := cmd.StderrPipe()
		go func() {
			io.Copy(os.Stderr, stderr)
		}()
		if err != nil {
			log.Printf("failed to get command stdout: %v", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err := cmd.Start(); err != nil {
			log.Printf("error starting command: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Fprintf(w, "%s\n", scanner.Text())
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if err := cmd.Wait(); err != nil {
			log.Printf("error running command: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	http.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean(r.URL.Path)
		if !strings.HasPrefix(p, "/images") {
			log.Printf("invalid path: %q", p)
			http.Error(w, "invalid path", http.StatusBadRequest)
		}
		f, err := os.Open(p)
		if err != nil {
			log.Printf("image not found: %q", p)
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		defer f.Close()
		io.Copy(w, f)
	})
}
func main() {
	http.ListenAndServe(":8080", nil)
}
