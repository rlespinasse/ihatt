package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestUseTUI(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		interactive bool
		want        bool
		wantErr     bool
	}{
		{"terminal by default", nil, true, true, false},
		{"plain text when not on a terminal", nil, false, false, false},
		{"--no-tui on a terminal", []string{"--no-tui"}, true, false, false},
		{"--no-tui when not on a terminal", []string{"--no-tui"}, false, false, false},
		{"--tui on a terminal", []string{"--tui"}, true, true, false},
		{"--tui when not on a terminal fails", []string{"--tui"}, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := isInteractive
			isInteractive = func() bool { return tt.interactive }
			t.Cleanup(func() { isInteractive = restore })

			c := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
			addOutputFlags(c)
			if err := c.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}

			got, err := useTUI(c)
			if (err != nil) != tt.wantErr {
				t.Fatalf("useTUI() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("useTUI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputFlagsAreExclusive(t *testing.T) {
	c := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	addOutputFlags(c)
	c.SetArgs([]string{"--tui", "--no-tui"})
	c.SilenceUsage = true
	c.SilenceErrors = true
	if err := c.Execute(); err == nil {
		t.Fatal("Execute() accepted both --tui and --no-tui")
	}
}
