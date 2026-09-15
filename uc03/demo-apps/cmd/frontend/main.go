// Frontend serves a page and proxies /api to the backend at BACKEND_URL.
package main

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/pr3s3nt/final_idp/uc03/demo-apps/internal/common"
)

func main() {
	common.Start("frontend")
	backend, err := url.Parse(common.Env("BACKEND_URL", true))
	if err != nil {
		panic(err)
	}
	title := os.Getenv("APP_TITLE")
	if title == "" {
		title = "Shop"
	}
	proxy := httputil.NewSingleHostReverseProxy(backend)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.Handle("/api/", proxy)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><title>%[1]s</title><h1>%[1]s</h1>
<p>frontend %[2]s, backend %[3]s</p>
<form onsubmit="fetch('/api/notes',{method:'POST',body:JSON.stringify({text:t.value})}).then(load);return false">
<input id=t><button>Add note</button></form><pre id=out></pre>
<script>function load(){fetch('/api/notes').then(r=>r.json()).then(j=>out.textContent=JSON.stringify(j,null,2))}load()</script>`,
			html.EscapeString(title), common.Version, html.EscapeString(backend.String()))
	})
	common.Serve("3000", mux)
}
