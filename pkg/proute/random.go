package proute

import "math/rand"

var (
	ASCIINum              = "0123456789"
	ASCIIHex              = "0123456789abcdefABCDEF"
	ASCIIPrintableNoSpace = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&\\'()*+,-./:;<=>?@[\\\\]^_`{|}~"
	ASCIISpecia           = "!\"#$%&\\'()*+,-./:;<=>?@[\\\\]^_`{|}~"
	ASCIIAlpha            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ASCIIAlphaNum         = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	DefaultCharSet        = ASCIINum
)

func RandomString(rng *rand.Rand, charset string, length int) string {
	if len(charset) == 0 {
		charset = ASCIINum
	}
	b := make([]rune, length)
	for i := range b {
		if rng != nil {
			b[i] = rune(charset[rng.Intn(len(charset))])
		} else {
			b[i] = rune(charset[rand.Intn(len(charset))])
		}
	}
	return string(b)
}
