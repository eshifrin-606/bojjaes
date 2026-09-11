package lineup

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var generationalSuffixes = []string{"Jr.", "Jr", "Sr.", "Sr", "II", "III", "IV", "V"}

func shortName(name string) string {
	words := strings.Fields(name)
	if len(words) < 2 {
		return name
	}
	if len(words) > 2 && slices.Contains(generationalSuffixes, words[len(words)-1]) {
		words = words[:len(words)-1]
	}
	lead := string([]rune(words[0])[0])
	if writtenAsInitials(words[0]) {
		lead = words[0]
	}
	return lead + " " + strings.Join(words[1:], " ")
}

// Only two capitals count as initials without a period: a longer all-capitals
// word ("JOE") is as likely to be a name typed in capitals.
func writtenAsInitials(word string) bool {
	if strings.Contains(word, ".") {
		return true
	}
	return utf8.RuneCountInString(word) == 2 &&
		!strings.ContainsFunc(word, func(r rune) bool { return !unicode.IsUpper(r) })
}

func fillShortNames(group []Record) {
	derived := make([]string, len(group))
	uses := make(map[string]int, len(group))
	for i, rec := range group {
		derived[i] = shortName(rec.Name)
		uses[derived[i]]++
	}
	for i := range group {
		group[i].ShortName = derived[i]
		if uses[derived[i]] > 1 {
			group[i].ShortName = group[i].Name
		}
	}
}
