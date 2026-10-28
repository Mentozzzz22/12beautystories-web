package admin

import (
	"strings"
	"unicode"
)

var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z",
	'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r",
	'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
	// Kazakh letters.
	'ә': "a", 'ғ': "g", 'қ': "k", 'ң': "n", 'ө': "o", 'ұ': "u", 'ү': "u", 'һ': "h", 'і': "i",
	// Common Latin diacritics in cosmetic names (Crème, Glacée).
	'à': "a", 'á': "a", 'â': "a", 'ä': "a", 'ç': "c", 'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'î': "i", 'ï': "i", 'ô': "o", 'ö': "o", 'ù': "u", 'û': "u", 'ü': "u",
}

const maxSlugLen = 80

// Slugify builds a URL slug from a name: "Энерджайзинг Shampoo 250 ml" → "enerdzhayzing-shampoo-250-ml".
func Slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		var part string
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			part = string(r)
		case translit[r] != "":
			part = translit[r]
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			continue
		default:
			if b.Len() > 0 {
				dash = true
			}
			continue
		}
		if dash {
			b.WriteByte('-')
			dash = false
		}
		b.WriteString(part)
	}
	s := b.String()
	if len(s) > maxSlugLen {
		s = strings.TrimRight(s[:maxSlugLen], "-")
	}
	return s
}

func validSlug(s string) bool {
	if s == "" || len(s) > maxSlugLen || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return !strings.Contains(s, "--")
}
