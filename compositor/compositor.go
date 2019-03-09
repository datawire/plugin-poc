package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

type ExampleCompositor struct{}

var (
	URLS string
)

func init() {
	val := os.Getenv("URLS")
	if val == "" {
		val = "http://httpbin.org/stream http://qotm.default/quote"
	}
	URLS = val
}

func (c *ExampleCompositor) SplitRequest(r *http.Request) ([]string, error) {
	var result []string
	for _, url := range strings.Fields(URLS) {
		result = append(result, resolve(url, r.URL.Path))
	}
	return result, nil
}

func resolve(base, relative string) string {
	relative = strings.TrimLeft(relative, "/")
	if relative == "" {
		return base
	} else {
		return fmt.Sprintf("%s/%s", base, relative)
	}
}

func (c *ExampleCompositor) JoinResponses(w http.ResponseWriter, r *http.Request, responses map[string]*http.Response) error {
	for _, response := range responses {
		_, err := io.Copy(w, response.Body)
		if err != nil {
			return err
		}
		response.Body.Close()
	}
	return nil
}

// Everything below this comment is generic and could be put into a
// library, the SplitRequest and JoinResponses methods contain any
// customizable business logic. You can try this by typing `go run
// compositor.go` and then running `curl localhost:8080`.

func main() {
	http.Handle("/", CompositorHandler(&ExampleCompositor{}))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type Compositor interface {
	SplitRequest(*http.Request) ([]string, error)
	JoinResponses(http.ResponseWriter, *http.Request, map[string]*http.Response) error
}

type compositorHandler struct {
	compositor Compositor
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%s [%p] ERROR: %s", r.RemoteAddr, r, err.Error())
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (h *compositorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urls, err := h.compositor.SplitRequest(r)
	if err != nil {
		handleError(w, r, err)
		return
	}

	log.Printf("%s [%p] splitting %s -> %s", r.RemoteAddr, r, r.URL, urls)

	var mu sync.Mutex
	responses := make(map[string]*http.Response)

	var group errgroup.Group
	for _, url := range urls {
		captured := url
		group.Go(func() error {
			resp, err := http.Get(captured)
			if err != nil {
				return err
			}
			mu.Lock()
			responses[captured] = resp
			mu.Unlock()
			return nil
		})
	}
	err = group.Wait()
	if err != nil {
		handleError(w, r, err)
		return
	}
	err = h.compositor.JoinResponses(w, r, responses)
	if err != nil {
		handleError(w, r, err)
		return
	}
}

func CompositorHandler(compositor Compositor) http.Handler {
	return &compositorHandler{compositor}
}
