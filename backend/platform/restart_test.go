package platform

import (
	"reflect"
	"testing"
)

func TestBuildRestartArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "no existing restart-delay",
			args: []string{"--hidden", "--auto-update=true"},
			want: []string{"--hidden", "--auto-update=true", "--restart-delay", "1s"},
		},
		{
			name: "empty args",
			args: []string{},
			want: []string{"--restart-delay", "1s"},
		},
		{
			name: "strips separate restart-delay flag and value",
			args: []string{"--hidden", "--restart-delay", "5s", "--auto-update=true"},
			want: []string{"--hidden", "--auto-update=true", "--restart-delay", "1s"},
		},
		{
			name: "strips equals-form restart-delay",
			args: []string{"--hidden", "--restart-delay=5s"},
			want: []string{"--hidden", "--restart-delay", "1s"},
		},
		{
			name: "strips trailing restart-delay without value",
			args: []string{"--hidden", "--restart-delay"},
			want: []string{"--hidden", "--restart-delay", "1s"},
		},
		{
			name: "does not consume a following option as the delay value",
			args: []string{"--restart-delay", "--auto-update=true"},
			want: []string{"--auto-update=true", "--restart-delay", "1s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRestartArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildRestartArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
