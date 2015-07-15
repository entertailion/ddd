package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andybons/gogif"
	"github.com/nfnt/resize"
)

func deepdreams(r io.Reader) (string, error) {
	cmd := exec.Command("python", "/ddd/deepdreams.py")
	dir, err := ioutil.TempDir("", "daydream")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v")
	}
	var buf bytes.Buffer
	io.Copy(&buf, r)

	cmd.Dir = dir
	cmd.Stdin = &buf
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error getting stdout pipe: %v")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("error getting stderr pipe: %v")
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting command: %v", err)
	}

	dream, err := filepath.Rel(os.TempDir(), dir)
	if err != nil {
		return "", fmt.Errorf("error getting temp directory: %v", err)
	}

	go func() {
		go func() {
			io.Copy(os.Stderr, stderr)
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Print(scanner.Text())
		}
		if err := cmd.Wait(); err != nil {
			log.Printf("deepdreams failed: %v", err)
			return
		}
		log.Printf("deepdreams finished success: %s", cmd.ProcessState.Success())
	}()

	dream = strings.TrimPrefix(dream, "daydream")
	return dream, nil
}

func init() {
	http.HandleFunc("/dream", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %q: %v", r.Method, r.URL.Path, r.Header)
		dream, err := deepdreams(r.Body)
		if err != nil {
			log.Printf("failed to dream: %v", err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Redirect(w, r, "/dream/"+dream, http.StatusFound)
		return
	})
	http.HandleFunc("/dream/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %q: %v", r.Method, r.URL.Path, r.Header)
		base := path.Base(r.URL.Path)
		ext := filepath.Ext(base)
		dream := strings.TrimSuffix(base, ext)
		dir, err := os.Open(filepath.Join(os.TempDir(), "daydream"+dream))
		if err != nil {
			log.Printf("dream not found: %q", dream)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer dir.Close()
		frames, err := dir.Readdir(-1)
		if err != nil {
			log.Printf("error reading dream %q: %v", dream, err)
			http.Error(w, fmt.Sprintf("error reading dream: %v", err.Error()), http.StatusNotFound)
			return
		}
		sort.Sort(ByCreation(frames))

		if ext == ".gif" {
			g := &gif.GIF{}
			cs := []chan *image.Paletted{}
			for _, f := range frames {
				fname := f.Name()
				if !strings.HasPrefix(fname, "frame-") || filepath.Ext(fname) != ".png" {
					continue
				}
				log.Printf("encoding: %q", fname)
				c := make(chan *image.Paletted)
				cs = append(cs, c)
				go func(frame string) {
					f, err := os.Open(filepath.Join(dir.Name(), frame))
					if err != nil {
						log.Printf("error reading dream %q frame %q: %v", dream, frame, err)
						c <- nil
						return
					}
					defer f.Close()
					img, err := png.Decode(f)
					if err != nil {
						log.Printf("error decoding dream %q frame %q: %v", dream, frame, err)
						c <- nil
						return
					}
					resized := resize.Resize(512, 0, img, resize.Lanczos3)
					c <- ImageToPaletted(resized)
					close(c)
				}(fname)
			}
			for i, c := range cs {
				img := <-c
				log.Printf("decoded frame: %d", i)
				if img == nil {
					http.Error(w, fmt.Sprintf("error decoding dream frame: %v", err.Error()), http.StatusInternalServerError)
					return
				}
				g.Image = append(g.Image, img)
				g.Delay = append(g.Delay, 20)
			}
			if err := gif.EncodeAll(w, g); err != nil {
				log.Printf("error encoding dream: %v", dream, err)
				http.Error(w, fmt.Sprintf("error encoding dream: %v", err.Error()), http.StatusInternalServerError)
			}
			return
		}

		var lastFrame = ""
		for i, f := range frames {
			log.Printf("frames %d: %q", i, f)
			fname := f.Name()
			if strings.HasPrefix(fname, "frame-") && filepath.Ext(fname) == ".png" {
				lastFrame = fname
			}
		}
		if lastFrame == "" {
			log.Printf("empty dream: %q", dream)
			http.Error(w, fmt.Sprintf("empty dream: %q", dream), http.StatusNotFound)
			return
		}
		f, err := os.Open(filepath.Join(dir.Name(), lastFrame))
		if err != nil {
			log.Printf("error opening dream frame %q: %v", lastFrame, err)
			http.Error(w, fmt.Sprintf("error opening dreamframe: %v", err.Error()), http.StatusNotFound)
		}
		defer f.Close()
		io.Copy(w, f)
	})
}
func main() {
	http.ListenAndServe(":8080", nil)
}

func ImageToPaletted(img image.Image) *image.Paletted {
	pm, ok := img.(*image.Paletted)
	if !ok {
		b := img.Bounds()
		pm = image.NewPaletted(b, nil)
		q := &gogif.MedianCutQuantizer{NumColor: 256}
		q.Quantize(pm, b, img, image.ZP)
	}
	return pm
}

type ByCreation []os.FileInfo

func (a ByCreation) Len() int           { return len(a) }
func (a ByCreation) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCreation) Less(i, j int) bool { return a[i].ModTime().UnixNano() < a[j].ModTime().UnixNano() }
