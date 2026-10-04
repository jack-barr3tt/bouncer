package api

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
)

type proxyPrefixKey struct{}

func (s *Server) proxyApp(c fiber.Ctx, upstream, prefix string) error {
	proxy, err := s.upstreamProxy(upstream)
	if err != nil {
		return s.internal(c, err)
	}
	handler := adaptor.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), proxyPrefixKey{}, prefix)))
	}))
	return handler(c)
}

func (s *Server) upstreamProxy(upstream string) (*httputil.ReverseProxy, error) {
	if existing, ok := s.proxies.Load(upstream); ok {
		return existing.(*httputil.ReverseProxy), nil
	}
	target, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = pr.In.Host
			pr.SetXForwarded()
			if proto := forwardedProto(pr.In); proto != "" {
				pr.Out.Header.Set("X-Forwarded-Proto", proto)
			}
			prefix, _ := pr.In.Context().Value(proxyPrefixKey{}).(string)
			if prefix != "" {
				pr.Out.Header.Set("X-Forwarded-Prefix", prefix)
			}
			if prefix != "" && prefix != "/" {
				pr.Out.URL.Path = stripPrefix(pr.Out.URL.Path, prefix)
				pr.Out.URL.RawPath = ""
			}
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "Bad gateway.", http.StatusBadGateway)
		},
		FlushInterval: -1,
	}
	actual, _ := s.proxies.LoadOrStore(upstream, proxy)
	return actual.(*httputil.ReverseProxy), nil
}

func forwardedProto(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "http" || proto == "https" {
		return proto
	}
	return ""
}

func stripPrefix(path, prefix string) string {
	if path == prefix {
		return "/"
	}
	if strings.HasPrefix(path, prefix+"/") {
		rest := strings.TrimPrefix(path, prefix)
		if rest == "" {
			return "/"
		}
		return rest
	}
	return path
}
