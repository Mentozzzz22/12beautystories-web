package admin

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Energizing Shampoo":      "energizing-shampoo",
		"Шампунь для объёма":      "shampun-dlya-obema",
		"Hydra 24+ Crème Glacée":  "hydra-24-creme-glacee",
		"  --Davines // MINU--  ": "davines-minu",
		"Қазақ өнімі":             "kazak-onimi",
		"日本 Shampoo":              "shampoo",
		"":                        "",
		"Расчёска (чёрная), 1 шт.": "rascheska-chernaya-1-sht",
	}
	for in, want := range tests {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugifyLength(t *testing.T) {
	got := Slugify(strings.Repeat("ab ", 60))
	if len(got) > maxSlugLen || strings.HasSuffix(got, "-") || !validSlug(got) {
		t.Errorf("Slugify(long) = %q (len %d)", got, len(got))
	}
}

func TestValidSlug(t *testing.T) {
	for s, want := range map[string]bool{
		"davines":      true,
		"hydra-24":     true,
		"":             false,
		"-davines":     false,
		"davines-":     false,
		"da--vines":    false,
		"Davines":      false,
		"davines minu": false,
		"шампунь":      false,
	} {
		if got := validSlug(s); got != want {
			t.Errorf("validSlug(%q) = %v, want %v", s, got, want)
		}
	}
}
