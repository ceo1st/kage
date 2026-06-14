package pack

import "os"

// Executable-format magic numbers, enough to tell the three desktop targets
// apart by their first bytes. We only need the family (windows, darwin, linux),
// not the architecture, so the smallest distinguishing prefix is plenty.
var (
	magicELF  = []byte{0x7f, 'E', 'L', 'F'}    // Linux, FreeBSD, and other ELF systems
	magicPE   = []byte{'M', 'Z'}               // Windows PE/COFF (DOS stub header)
	machOLE64 = []byte{0xcf, 0xfa, 0xed, 0xfe} // Mach-O 64-bit, little-endian (amd64, arm64)
	machOLE32 = []byte{0xce, 0xfa, 0xed, 0xfe} // Mach-O 32-bit, little-endian
	machOBE64 = []byte{0xfe, 0xed, 0xfa, 0xcf} // Mach-O 64-bit, big-endian
	machOBE32 = []byte{0xfe, 0xed, 0xfa, 0xce} // Mach-O 32-bit, big-endian
	machOFat  = []byte{0xca, 0xfe, 0xba, 0xbe} // Mach-O universal (fat) binary
)

// SniffOS reads the first bytes of an executable and returns the GOOS family it
// was built for: "windows", "darwin", "linux", or "" when the bytes match none
// of them. It is how pack decides whether a cross-built viewer needs a .exe
// suffix and which run hint to print, without trusting the base's file name.
func SniffOS(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	head := make([]byte, 4)
	if _, err := f.ReadAt(head, 0); err != nil {
		return ""
	}
	switch {
	case hasPrefix(head, magicPE):
		return "windows"
	case hasPrefix(head, magicELF):
		return "linux"
	case hasPrefix(head, machOLE64), hasPrefix(head, machOLE32),
		hasPrefix(head, machOBE64), hasPrefix(head, machOBE32),
		hasPrefix(head, machOFat):
		return "darwin"
	default:
		return ""
	}
}

func hasPrefix(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := range prefix {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}
