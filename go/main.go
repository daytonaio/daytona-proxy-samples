package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/patrickmn/go-cache"
)

type Config struct {
	DaytonaAPIURL string
	DaytonaAPIKey string
	Port          string
}

//go:embed error.html
var errorPageHTML string

type Proxy struct {
	proxy     *httputil.ReverseProxy
	cache     *cache.Cache
	apiClient *http.Client
	config    *Config
}

var (
	sandboxIDRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	portRegex      = regexp.MustCompile(`^[0-9]+$`)
)

type PreviewResponse struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func validateInputs(sandboxID, port string) error {
	if sandboxID == "" || port == "" {
		return fmt.Errorf("sandbox ID and port cannot be empty")
	}
	if !sandboxIDRegex.MatchString(sandboxID) || !portRegex.MatchString(port) {
		return fmt.Errorf("invalid format")
	}
	return nil
}

func NewProxy(config *Config) *Proxy {
	p := &Proxy{
		cache:     cache.New(2*time.Minute, 5*time.Minute),
		config:    config,
		apiClient: &http.Client{Timeout: 10 * time.Second},
	}

	p.proxy = &httputil.ReverseProxy{
		Director: p.director,
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode != http.StatusOK {
				return p.serveErrorPage(resp)
			}

			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			p.writeErrorPage(w)
		},
	}

	return p
}

func (p *Proxy) serveErrorPage(resp *http.Response) error {
	resp.Body.Close()
	resp.StatusCode = http.StatusBadGateway
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(errorPageHTML)))
	resp.Header.Del("Content-Encoding")
	resp.Body = io.NopCloser(strings.NewReader(errorPageHTML))
	return nil
}

func (p *Proxy) writeErrorPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	w.Write([]byte(errorPageHTML))
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	sandboxID, port := getSandboxIdAndPortFromUrl(r.Host)
	if err := validateInputs(sandboxID, port); err != nil {
		log.Printf("Invalid request: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	p.proxy.ServeHTTP(w, r)
}

func (p *Proxy) director(req *http.Request) {
	sandboxId, port := getSandboxIdAndPortFromUrl(req.Host)

	preview, err := p.getPreview(req.Context(), sandboxId, port)
	if err != nil {
		log.Printf("Failed to get preview: %v", err)
		req.URL.Host = "invalid.local"
		return
	}

	targetUrl, err := url.Parse(preview.URL)
	if err != nil {
		log.Printf("Invalid target URL: %v", err)
		req.URL.Host = "invalid.local"
		return
	}

	req.URL.Scheme = targetUrl.Scheme
	req.URL.Host = targetUrl.Host
	req.URL.Path = singleJoiningSlash(targetUrl.Path, req.URL.Path)
	req.Host = targetUrl.Host
	req.Header.Set("X-Daytona-Preview-Token", preview.Token)
}

func (p *Proxy) getPreview(ctx context.Context, sandboxId, port string) (*PreviewResponse, error) {
	cacheKey := fmt.Sprintf("%s-%s", sandboxId, port)

	if x, found := p.cache.Get(cacheKey); found {
		return x.(*PreviewResponse), nil
	}

	apiUrl := fmt.Sprintf("%s/sandbox/%s/ports/%s/preview-url", p.config.DaytonaAPIURL, sandboxId, port)
	req, err := http.NewRequestWithContext(ctx, "GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.config.DaytonaAPIKey)

	resp, err := p.apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var preview PreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		return nil, err
	}

	p.cache.Set(cacheKey, &preview, cache.DefaultExpiration)
	return &preview, nil
}

func loadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	config := &Config{
		DaytonaAPIURL: os.Getenv("DAYTONA_API_URL"),
		DaytonaAPIKey: os.Getenv("DAYTONA_API_KEY"),
		Port:          os.Getenv("PORT"),
	}

	if config.Port == "" {
		config.Port = "3000"
	}

	if config.DaytonaAPIURL == "" || config.DaytonaAPIKey == "" {
		log.Fatal("DAYTONA_API_URL and DAYTONA_API_KEY must be set")
	}

	return config
}

func main() {
	config := loadConfig()

	proxy := NewProxy(config)

	server := &http.Server{
		Addr:           ":" + config.Port,
		Handler:        proxy,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting proxy server on port %s", config.Port)
		log.Printf("Daytona API URL: %s", config.DaytonaAPIURL)
		log.Printf("Server ready to accept connections")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server gracefully stopped")
	}
}

func getSandboxIdAndPortFromUrl(host string) (string, string) {
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	parts := strings.Split(host, ".")
	if len(parts) < 1 {
		return "", ""
	}

	subdomain := parts[0]
	subdomainParts := strings.SplitN(subdomain, "-", 2)
	if len(subdomainParts) != 2 {
		return "", ""
	}

	port := subdomainParts[0]
	sandboxId := subdomainParts[1]

	return sandboxId, port
}

func singleJoiningSlash(a, b string) string {
	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/")
	switch {
	case aSlash && bSlash:
		return a + b[1:]
	case !aSlash && !bSlash:
		return a + "/" + b
	}
	return a + b
}
