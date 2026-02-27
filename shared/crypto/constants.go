package crypto

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"0123456789"

const charsetLength = byte(len(charset)) // = 62

// maxUnbiased is the largest multiple of 62 less than 256.
// Values >= maxUnbiased are rejected to avoid modulo bias.
const maxUnbiased = byte(256 - (256 % int(charsetLength))) // 248
