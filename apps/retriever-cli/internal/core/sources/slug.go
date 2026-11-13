package sources

import "strings"

// slug turns a document title into a file name stem. It keeps ASCII letters and digits,
// lowercased, joins each run of other characters with one hyphen, and trims the ends.
// "Getting Started" becomes "getting-started". A title with no letters or digits gives "".
func slug(title string) string {
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			pendingHyphen = false
			continue
		}
		pendingHyphen = true
	}
	return b.String()
}
