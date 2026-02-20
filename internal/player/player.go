package player

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"time"
)

// TODO: use unique path per session when IPC is needed.
const ipcSocket = "/tmp/ani-tui-mpv.sock"

// Session manages an mpv process and the localhost HTTP server that feeds it.
type Session struct {
	cmd    *exec.Cmd
	server *http.Server
	done   chan error
}

// Start launches a localhost HTTP proxy serving the reader, then starts mpv
// pointed at the proxy URL. The mpvPath may be empty to use "mpv" from PATH.
func Start(mpvPath string, reader io.ReadSeeker, filename string) (*Session, error) {
	if mpvPath == "" {
		mpvPath = "mpv"
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/video", func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, filename, time.Time{}, reader)
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)

	url := fmt.Sprintf("http://127.0.0.1:%d/video", ln.Addr().(*net.TCPAddr).Port)

	cmd := exec.Command(mpvPath,
		"--input-ipc-server="+ipcSocket,
		"--force-window=yes",
		url,
	)

	if err := cmd.Start(); err != nil {
		srv.Close()
		return nil, fmt.Errorf("start mpv: %w", err)
	}

	s := &Session{
		cmd:    cmd,
		server: srv,
		done:   make(chan error, 1),
	}

	go func() {
		s.done <- cmd.Wait()
	}()

	return s, nil
}

// Wait returns a channel that receives when the mpv process exits.
func (s *Session) Wait() <-chan error {
	return s.done
}

// Close kills mpv if still running, shuts down the HTTP server, and waits for
// the process goroutine to finish.
func (s *Session) Close() {
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		// Drain done channel so the goroutine can exit.
		select {
		case <-s.done:
		case <-time.After(3 * time.Second):
		}
	}

	if s.server != nil {
		s.server.Close()
	}
}
