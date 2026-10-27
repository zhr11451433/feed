package observability

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof" // 【未学】pprof
	"time"
)

type PprofServer struct {
	name            string
	server          *http.Server
	shutdownTimeout time.Duration
}

func NewPprofMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

// 【未学】enabled 为 false 时不听端口。Listen 成功后立刻返回，Serve 在后台跑。

func NewPprofServer(name string, enabled bool, addr string) (*PprofServer, error) {
	if !enabled || addr == "" {
		return nil, nil
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start %s pprof server on %s: %w", name, addr, err)
	}
	pprofServer := &PprofServer{
		name:            name,
		shutdownTimeout: 3 * time.Second,
	}
	pprofServer.server = &http.Server{
		Addr:              addr,
		Handler:           NewPprofMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("%s pprof listening on %s", name, addr)
		if err := pprofServer.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("%s pprof server error: %v", name, err)
		}
	}()
	return pprofServer, nil
}

func (s *PprofServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown %s pprof server: %v", s.name, err)
		return err
	}
	return nil
}
