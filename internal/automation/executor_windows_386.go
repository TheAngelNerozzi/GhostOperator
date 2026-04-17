//go:build windows && 386

package automation

// inputSize386 is the correct size of the Windows INPUT structure on 32-bit (386):
// uint32 type (4) + union (24) = 28 bytes (no padding needed on 32-bit)
const inputSize = 28
