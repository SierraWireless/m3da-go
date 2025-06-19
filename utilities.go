package m3da

import (
	"fmt"
)

// hexDump prints byte data in hexdump -C format
func hexDump(data []byte) {
	const bytesPerLine = 16

	for i := 0; i < len(data); i += bytesPerLine {
		// Print offset
		fmt.Printf("%08x  ", i)

		// Print hex bytes in two groups of 8
		for j := 0; j < bytesPerLine; j++ {
			if i+j < len(data) {
				fmt.Printf("%02x ", data[i+j])
			} else {
				fmt.Print("   ") // padding for incomplete lines
			}

			// Add extra space after 8th byte
			if j == 7 {
				fmt.Print(" ")
			}
		}

		// Print ASCII representation
		fmt.Print(" |")
		for j := 0; j < bytesPerLine && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 { // printable ASCII
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Print("|\n")
	}

	// Print final offset
	fmt.Printf("%08x\n", len(data))
}

// Print a byte array with a label
func printHexDump(data []byte, label string) {
	if label != "" {
		fmt.Printf("%s (%d bytes):\n", label, len(data))
	}
	hexDump(data)
}

// Convert a generic slice into an interface{} slice/array
// mostly used for handling the various List format types
func convertSliceToInterface[T any](slice []T) []interface{} {
	list := make([]interface{}, len(slice))
	for i, val := range slice {
		list[i] = val
	}
	return list
}
