// Package console implements the framed protocol on the local supervisor
// Unix socket. Framing: 4-byte big-endian length + JSON message. Message
// sizes are limited; slow clients are disconnected without blocking Minecraft.
package console

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Chansovisoth/Bonghos/internal/config"
	"github.com/Chansovisoth/Bonghos/internal/supervisor"
)

const maxFrame = 64 * 1024

// Message types exchanged on the socket.
type Message struct {
	Type    string   `json:"type"` // hello | history | line | internal_line | status | command | internal_command | error | ok
	Line    string   `json:"line,omitempty"`
	Lines   []string `json:"lines,omitempty"`
	State   string   `json:"state,omitempty"`
	Command string   `json:"command,omitempty"`
	Error   string   `json:"error,omitempty"`
	Auth    string   `json:"auth,omitempty"`
}

func writeFrame(w io.Writer, m *Message) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if len(data) > maxFrame {
		return errors.New("frame too large")
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// historyChunkBytes keeps each replayed history frame well inside maxFrame,
// leaving room for the JSON envelope around the lines.
const historyChunkBytes = maxFrame / 2

// writeHistory replays buffered console lines in frames small enough to send.
// A modded server's boot output does not fit in one frame: sending it as a
// single message meant the write failed and the client silently received no
// backlog at all. An over-long individual line is truncated rather than
// dropping its chunk, because slightly clipped output beats an empty console.
func writeHistory(conn net.Conn, history []string) error {
	if len(history) == 0 {
		return writeFrame(conn, &Message{Type: "history", Lines: nil})
	}
	chunk := make([]string, 0, 64)
	size := 0
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		err := writeFrame(conn, &Message{Type: "history", Lines: chunk})
		chunk = chunk[:0]
		size = 0
		return err
	}
	for _, line := range history {
		if len(line) > historyChunkBytes {
			line = line[:historyChunkBytes-16] + " …[truncated]"
		}
		if size+len(line)+8 > historyChunkBytes {
			if err := flush(); err != nil {
				return err
			}
		}
		chunk = append(chunk, line)
		size += len(line) + 8 // rough JSON overhead per element
	}
	return flush()
}

func readFrame(r io.Reader) (*Message, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > maxFrame {
		return nil, errors.New("frame too large")
	}
	data := make([]byte, n)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ----- server ----------------------------------------------------------------

// Server exposes the supervisor console on the runtime Unix socket.
type Server struct {
	Home string
	Sup  *supervisor.Supervisor

	ln   net.Listener
	auth string // shared token written next to the socket, mode 0600
	mu   sync.Mutex
}

// Start creates the socket (0600, supervisor lifetime only) and serves.
func (s *Server) Start() error {
	sockPath := filepath.Join(s.Home, config.FileSupSocket)
	// remove verified stale socket
	if _, err := os.Stat(sockPath); err == nil {
		if conn, err := net.DialTimeout("unix", sockPath, time.Second); err == nil {
			conn.Close()
			return fmt.Errorf("another supervisor is already listening on %s", sockPath)
		}
		os.Remove(sockPath)
	}
	// per-run auth token: even local clients must present it
	tok := make([]byte, 16)
	if _, err := readRand(tok); err != nil {
		return err
	}
	s.auth = fmt.Sprintf("%x", tok)
	tokPath := sockPath + ".token"
	if err := os.WriteFile(tokPath, []byte(s.auth), 0o600); err != nil {
		return err
	}
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return err
	}
	os.Chmod(sockPath, 0o600)
	s.ln = ln
	go s.acceptLoop()
	return nil
}

func readRand(b []byte) (int, error) {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.ReadFull(f, b)
}

func (s *Server) Close() {
	if s.ln != nil {
		s.ln.Close()
	}
	sockPath := filepath.Join(s.Home, config.FileSupSocket)
	os.Remove(sockPath)
	os.Remove(sockPath + ".token")
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	hello, err := readFrame(conn)
	if err != nil || hello.Type != "hello" || hello.Auth != s.auth {
		writeFrame(conn, &Message{Type: "error", Error: "authentication failed"})
		return
	}
	conn.SetReadDeadline(time.Time{})

	history, lines, cancel := s.Sup.Subscribe()
	defer cancel()
	internalLines, cancelInternal := s.Sup.SubscribeInternal()
	defer cancelInternal()

	writeFrame(conn, &Message{Type: "status", State: string(s.Sup.State())})

	// History is replayed in chunks. A single frame is capped, and a modded
	// server's boot output does not fit in one: sending it as one frame meant
	// the write failed and the client silently received no backlog at all.
	if err := writeHistory(conn, history); err != nil {
		return
	}

	// reader: console command input (only Minecraft commands, never a shell)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			m, err := readFrame(conn)
			if err != nil {
				return
			}
			switch m.Type {
			case "command":
				if err := s.Sup.SendCommand(m.Command); err != nil {
					writeFrame(conn, &Message{Type: "error", Error: err.Error()})
				}
			case "internal_command":
				// Bonghos bookkeeping (player polling). The supervisor only
				// hides commands on its own closed list, so this cannot be
				// used to run something invisibly.
				if err := s.Sup.SendInternalCommand(m.Command); err != nil {
					writeFrame(conn, &Message{Type: "error", Error: err.Error()})
				}
			case "control":
				// Fixed lifecycle verbs only — never a shell.
				var err error
				switch m.Command {
				case "stop":
					err = s.Sup.Stop(true)
				case "restart":
					err = s.Sup.Restart()
				case "force-stop":
					err = s.Sup.ForceStop()
				default:
					err = fmt.Errorf("unknown control verb %q", m.Command)
				}
				if err != nil {
					writeFrame(conn, &Message{Type: "error", Error: err.Error()})
				} else {
					writeFrame(conn, &Message{Type: "ok"})
				}
			}
		}
	}()

	// writer: live lines with backpressure (drop-slow behavior is upstream)
	for {
		select {
		case <-done:
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := writeFrame(conn, &Message{Type: "line", Line: line}); err != nil {
				return // slow or dead client: disconnect without blocking Minecraft
			}
		case line, ok := <-internalLines:
			if !ok {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := writeFrame(conn, &Message{Type: "internal_line", Line: line}); err != nil {
				return
			}
		}
	}
}

// ----- client ----------------------------------------------------------------

// Client connects the current terminal (or another consumer) to the supervisor.
type Client struct {
	Home string
	conn net.Conn
}

// Dial connects and authenticates using the socket token.
func Dial(home string) (*Client, error) {
	sockPath := filepath.Join(home, config.FileSupSocket)
	tok, err := os.ReadFile(sockPath + ".token")
	if err != nil {
		return nil, errors.New("no active supervisor console is available (is Minecraft running?)")
	}
	conn, err := net.DialTimeout("unix", sockPath, 3*time.Second)
	if err != nil {
		return nil, errors.New("no active supervisor console is available (is Minecraft running?)")
	}
	c := &Client{Home: home, conn: conn}
	if err := writeFrame(conn, &Message{Type: "hello", Auth: strings.TrimSpace(string(tok))}); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() { c.conn.Close() }

// Read returns the next message from the supervisor.
func (c *Client) Read() (*Message, error) { return readFrame(c.conn) }

// Send forwards a Minecraft console command.
// Control sends a lifecycle verb (stop | restart | force-stop) to the supervisor.
func (c *Client) Control(verb string) error {
	return writeFrame(c.conn, &Message{Type: "control", Command: verb})
}

func (c *Client) Send(cmd string) error {
	return writeFrame(c.conn, &Message{Type: "command", Command: cmd})
}

// SendInternal issues a Bonghos bookkeeping command whose echo and reply are
// kept out of the operator's console.
func (c *Client) SendInternal(cmd string) error {
	return writeFrame(c.conn, &Message{Type: "internal_command", Command: cmd})
}

// Interactive attaches stdin/stdout of the current terminal to the console.
func (c *Client) Interactive(stdin io.Reader, stdout io.Writer) error {
	go func() {
		sc := newLineScanner(stdin)
		for sc.Scan() {
			line := sc.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			c.Send(line)
		}
	}()
	for {
		m, err := c.Read()
		if err != nil {
			return nil // supervisor went away (server stopped)
		}
		switch m.Type {
		case "history":
			for _, l := range m.Lines {
				fmt.Fprintln(stdout, l)
			}
		case "line":
			fmt.Fprintln(stdout, m.Line)
		case "status":
			fmt.Fprintf(stdout, "[bonghos] server state: %s\n", m.State)
		case "error":
			fmt.Fprintf(stdout, "[bonghos] %s\n", m.Error)
		}
	}
}

func newLineScanner(r io.Reader) *lineScanner { return &lineScanner{r: r} }

type lineScanner struct {
	r    io.Reader
	buf  []byte
	line string
}

func (s *lineScanner) Scan() bool {
	var out []byte
	one := make([]byte, 1)
	for {
		n, err := s.r.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				s.line = string(out)
				return true
			}
			out = append(out, one[0])
			if len(out) > 4096 {
				s.line = string(out)
				return true
			}
			continue
		}
		if err != nil {
			if len(out) > 0 {
				s.line = string(out)
				return true
			}
			return false
		}
	}
}

func (s *lineScanner) Text() string { return s.line }
