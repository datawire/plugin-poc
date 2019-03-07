package main

import (
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/sync/errgroup"
)

type ExampleCompositor struct{}

func (c *ExampleCompositor) SplitRequest(r *http.Request) ([]string, error) {
	return []string{"http://httpbin.org/headers", "http://httpbin.org/ip"}, nil
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

func (h *compositorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urls, err := h.compositor.SplitRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = h.compositor.JoinResponses(w, r, responses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CompositorHandler(compositor Compositor) http.Handler {
	return &compositorHandler{compositor}
}
