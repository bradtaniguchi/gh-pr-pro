package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/term"
)

// DefaultFrames provides standard Braille dot animation frames.
var DefaultFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// DefaultInterval is the standard animation frame interval.
const DefaultInterval = 85 * time.Millisecond

// Option configures a Spinner instance.
type Option func(*options)

type options struct {
	writer   io.Writer
	frames   []string
	interval time.Duration
	enabled  *bool
}

// WithWriter sets a custom output writer for the spinner.
func WithWriter(w io.Writer) Option {
	return func(o *options) {
		if w != nil {
			o.writer = w
		}
	}
}

// WithEnabled explicitly enables or disables the spinner.
func WithEnabled(enabled bool) Option {
	return func(o *options) {
		o.enabled = &enabled
	}
}

// WithInterval sets the animation frame duration.
func WithInterval(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.interval = d
		}
	}
}

// WithFrames sets custom animation frames for the spinner.
func WithFrames(frames []string) Option {
	return func(o *options) {
		if len(frames) > 0 {
			o.frames = frames
		}
	}
}

// Spinner provides a thread-safe terminal spinner for long-running operations.
type Spinner struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	writer   io.Writer
	frames   []string
	interval time.Duration
	msg      string
	enabled  bool
	active   bool
	stopCh   chan struct{}
}

// NewSpinner creates a new Spinner initialized with the given message and optional configurations.
func NewSpinner(msg string, opts ...Option) *Spinner {
	optsConfig := &options{
		writer:   os.Stderr,
		frames:   DefaultFrames,
		interval: DefaultInterval,
	}

	for _, opt := range opts {
		opt(optsConfig)
	}

	var enabled bool
	if optsConfig.enabled != nil {
		enabled = *optsConfig.enabled
	} else {
		enabled = isInteractive(optsConfig.writer)
	}

	return &Spinner{
		writer:   optsConfig.writer,
		frames:   optsConfig.frames,
		interval: optsConfig.interval,
		msg:      msg,
		enabled:  enabled,
	}
}

// isInteractive checks if the writer is attached to an interactive terminal
// and that terminal environment variables do not suppress interactive UI.
func isInteractive(w io.Writer) bool {
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("CI") != "" {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(f)
	}

	return false
}

// Start initiates the spinner animation in a background goroutine if enabled and not already running.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled || s.active {
		return
	}

	s.active = true
	s.stopCh = make(chan struct{})
	s.wg.Add(1)

	go s.run(s.stopCh)
}

// run is the background loop rendering animation frames.
func (s *Spinner) run(stopCh <-chan struct{}) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frameIdx := 0
	s.render(frameIdx)

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			frameIdx = (frameIdx + 1) % len(s.frames)
			s.render(frameIdx)
		}
	}
}

// render writes a single spinner frame and message to the configured writer.
func (s *Spinner) render(frameIdx int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	frame := s.frames[frameIdx%len(s.frames)]
	if s.msg != "" {
		fmt.Fprintf(s.writer, "\r\033[2K%s %s", frame, s.msg)
	} else {
		fmt.Fprintf(s.writer, "\r\033[2K%s", frame)
	}
}

// SetMessage updates the current status message displayed beside the spinner.
func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msg = msg
}

// Stop halts the spinner animation, clears the current terminal line, and resets state.
// Safe to call multiple times or on unstarted spinners.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}

	s.active = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enabled {
		fmt.Fprint(s.writer, "\r\033[2K\r")
	}
}

// Active returns whether the spinner animation is currently running.
func (s *Spinner) Active() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Enabled returns whether the spinner is enabled for terminal output.
func (s *Spinner) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}
