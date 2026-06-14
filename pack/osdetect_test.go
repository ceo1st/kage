package pack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSniffOS(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want string
	}{
		{"elf", []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01}, "linux"},
		{"pe", []byte{'M', 'Z', 0x90, 0x00}, "windows"},
		{"macho-le64", []byte{0xcf, 0xfa, 0xed, 0xfe}, "darwin"},
		{"macho-le32", []byte{0xce, 0xfa, 0xed, 0xfe}, "darwin"},
		{"macho-be64", []byte{0xfe, 0xed, 0xfa, 0xcf}, "darwin"},
		{"macho-fat", []byte{0xca, 0xfe, 0xba, 0xbe}, "darwin"},
		{"unknown", []byte{0x00, 0x01, 0x02, 0x03}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), tc.name)
			if err := os.WriteFile(p, tc.head, 0o644); err != nil {
				t.Fatal(err)
			}
			if got := SniffOS(p); got != tc.want {
				t.Errorf("SniffOS(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestSniffOSMissingFile(t *testing.T) {
	if got := SniffOS(filepath.Join(t.TempDir(), "nope")); got != "" {
		t.Errorf("SniffOS(missing) = %q, want empty", got)
	}
}
