package ui

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewSpinner_Options(t *testing.T) {
	customFrames := []string{".", "o", "O", "o"}
	customInterval := 50 * time.Millisecond
	buf := &bytes.Buffer{}

	tests := []struct {
		name         string
		msg          string
		opts         []Option
		wantInterval time.Duration
		wantFrames   []string
		wantEnabled  bool
	}{
		{
			name:         "default options disabled on buffer by default",
			msg:          "Loading...",
			opts:         []Option{WithWriter(buf)},
			wantInterval: DefaultInterval,
			wantFrames:   DefaultFrames,
			wantEnabled:  false,
		},
		{
			name: "custom options with explicit enabled",
			msg:  "Fetching PRs...",
			opts: []Option{
				WithWriter(buf),
				WithEnabled(true),
				WithInterval(customInterval),
				WithFrames(customFrames),
			},
			wantInterval: customInterval,
			wantFrames:   customFrames,
			wantEnabled:  true,
		},
		{
			name: "explicitly disabled",
			msg:  "Silent task",
			opts: []Option{
				WithWriter(buf),
				WithEnabled(false),
			},
			wantInterval: DefaultInterval,
			wantFrames:   DefaultFrames,
			wantEnabled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSpinner(tt.msg, tt.opts...)
			if s.Enabled() != tt.wantEnabled {
				t.Errorf("Enabled() = %v, want %v", s.Enabled(), tt.wantEnabled)
			}
			if s.interval != tt.wantInterval {
				t.Errorf("interval = %v, want %v", s.interval, tt.wantInterval)
			}
			if len(s.frames) != len(tt.wantFrames) {
				t.Errorf("frames len = %v, want %v", len(s.frames), len(tt.wantFrames))
			}
		})
	}
}

func TestSpinner_StartUpdateStop(t *testing.T) {
	tests := []struct {
		name           string
		initialMsg     string
		updatedMsg     string
		interval       time.Duration
		sleepBeforeMsg time.Duration
		sleepAfterMsg  time.Duration
	}{
		{
			name:           "basic start and stop",
			initialMsg:     "Loading data...",
			updatedMsg:     "Processing results...",
			interval:       20 * time.Millisecond,
			sleepBeforeMsg: 45 * time.Millisecond,
			sleepAfterMsg:  45 * time.Millisecond,
		},
		{
			name:           "empty initial message",
			initialMsg:     "",
			updatedMsg:     "Now has text",
			interval:       15 * time.Millisecond,
			sleepBeforeMsg: 30 * time.Millisecond,
			sleepAfterMsg:  30 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			s := NewSpinner(tt.initialMsg,
				WithWriter(buf),
				WithEnabled(true),
				WithInterval(tt.interval),
			)

			if s.Active() {
				t.Fatal("Active() = true before Start()")
			}

			s.Start()
			if !s.Active() {
				t.Fatal("Active() = false after Start()")
			}

			// Calling Start() again should be a no-op
			s.Start()

			time.Sleep(tt.sleepBeforeMsg)
			s.SetMessage(tt.updatedMsg)
			time.Sleep(tt.sleepAfterMsg)

			s.Stop()
			if s.Active() {
				t.Fatal("Active() = true after Stop()")
			}

			out := buf.String()
			if tt.initialMsg != "" && !strings.Contains(out, tt.initialMsg) {
				t.Errorf("output does not contain initial message %q, got: %q", tt.initialMsg, out)
			}
			if !strings.Contains(out, tt.updatedMsg) {
				t.Errorf("output does not contain updated message %q, got: %q", tt.updatedMsg, out)
			}
			if !strings.HasSuffix(out, "\r\033[2K\r") {
				t.Errorf("output does not end with clear sequence, got: %q", out)
			}
		})
	}
}

func TestSpinner_Disabled(t *testing.T) {
	buf := &bytes.Buffer{}
	s := NewSpinner("Disabled spinner",
		WithWriter(buf),
		WithEnabled(false),
		WithInterval(10*time.Millisecond),
	)

	s.Start()
	if s.Active() {
		t.Fatal("Active() should be false for disabled spinner")
	}

	s.SetMessage("Updated message")
	time.Sleep(30 * time.Millisecond)

	s.Stop()
	if s.Active() {
		t.Fatal("Active() should be false after Stop()")
	}

	if buf.Len() != 0 {
		t.Fatalf("expected no output for disabled spinner, got %q", buf.String())
	}
}

func TestSpinner_MultipleStops(t *testing.T) {
	t.Run("stop unstarted spinner", func(t *testing.T) {
		buf := &bytes.Buffer{}
		s := NewSpinner("Unstarted", WithWriter(buf), WithEnabled(true))
		// Should safely no-op
		s.Stop()
		s.Stop()
		if s.Active() {
			t.Error("Active() should be false")
		}
		if buf.Len() != 0 {
			t.Errorf("expected empty buffer, got %q", buf.String())
		}
	})

	t.Run("stop already stopped spinner", func(t *testing.T) {
		buf := &bytes.Buffer{}
		s := NewSpinner("Running",
			WithWriter(buf),
			WithEnabled(true),
			WithInterval(10*time.Millisecond),
		)
		s.Start()
		time.Sleep(25 * time.Millisecond)
		s.Stop()
		if s.Active() {
			t.Error("Active() should be false after first stop")
		}
		// Subsequent stops should not panic or corrupt
		s.Stop()
		s.Stop()
	})
}

func TestSpinner_ConcurrentRace(t *testing.T) {
	buf := &bytes.Buffer{}
	s := NewSpinner("Initial",
		WithWriter(buf),
		WithEnabled(true),
		WithInterval(5*time.Millisecond),
	)

	s.Start()

	var wg sync.WaitGroup
	// Concurrently call SetMessage
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				s.SetMessage("Concurrent update")
				_ = s.Active()
				_ = s.Enabled()
			}
		}(i)
	}

	// Concurrently call Stop and Start
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			s.Stop()
		}()
	}

	wg.Wait()
	s.Stop() // Final cleanup
}

func TestIsInteractive_EnvVars(t *testing.T) {
	tests := []struct {
		name      string
		envKey    string
		envVal    string
		wantIsTTY bool
	}{
		{
			name:      "TERM=dumb disables interactive",
			envKey:    "TERM",
			envVal:    "dumb",
			wantIsTTY: false,
		},
		{
			name:      "CI=true disables interactive",
			envKey:    "CI",
			envVal:    "true",
			wantIsTTY: false,
		},
		{
			name:      "NO_COLOR=1 disables interactive",
			envKey:    "NO_COLOR",
			envVal:    "1",
			wantIsTTY: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origVal, exists := os.LookupEnv(tt.envKey)
			defer func() {
				if exists {
					_ = os.Setenv(tt.envKey, origVal)
				} else {
					_ = os.Unsetenv(tt.envKey)
				}
			}()

			_ = os.Setenv(tt.envKey, tt.envVal)

			buf := &bytes.Buffer{}
			if isInteractive(buf) != tt.wantIsTTY {
				t.Errorf("isInteractive() = %v, want %v for %s=%s", isInteractive(buf), tt.wantIsTTY, tt.envKey, tt.envVal)
			}
		})
	}
}
