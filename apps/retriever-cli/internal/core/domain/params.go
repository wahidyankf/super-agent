package domain

import (
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Limits of the parameter kinds (ADR 007 section 2, item 7). A catalog row names a kind, and the kind's validator
// applies these limits, so a limit is written once.
const (
	// MaxIDLength is the most bytes any ID kind may hold: an ID is a path segment of the API, and every kind shares this bound.
	MaxIDLength = 255
	// MaxQueryLength is the most characters a search query may hold.
	MaxQueryLength = 1000
	// MaxTextBytes is the most bytes a Text parameter may hold: 1 MiB.
	MaxTextBytes = 1 << 20
	// MaxNameLength is the most characters a Name parameter may hold.
	MaxNameLength = 255
	// MaxIndexLength is the most characters a fractional index may hold (the idx notation of the inventory).
	MaxIndexLength = 256
	// MaxIDList is the most IDs an IDList parameter may hold.
	MaxIDList = 50
	// DefaultPageLimit is the page size of a list operation that names none.
	DefaultPageLimit = 25
	// MaxPageLimit is the largest page a list operation may ask for. The server refuses a larger one with 400.
	MaxPageLimit = 100
)

// The patterns the validators apply. The ID pattern is the one the source API accepts in a path.
var (
	idPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// shareSlugPattern is the slug form of a share ID that the server accepts (zodShareIdType).
	shareSlugPattern = regexp.MustCompile(`^[0-9a-z-]+$`)
)

// ValidateID accepts a non-empty identifier of letters, digits, "_" and "-" (ADR 001).
func ValidateID(raw string) error {
	if len(raw) > MaxIDLength || !idPattern.MatchString(raw) {
		return ErrInvalidID
	}
	return nil
}

// ValidateUUID accepts a canonical UUID, with hyphens, in either letter case. The server refuses a urlId slug here
// with 400, so the check is local.
func ValidateUUID(raw string) error {
	if !uuidPattern.MatchString(raw) {
		return ErrInvalidUUID
	}
	return nil
}

// ValidateShareID accepts a share ID: a canonical UUID, or a slug of lower-case letters, digits, and hyphens.
func ValidateShareID(raw string) error {
	if ValidateUUID(raw) == nil || (len(raw) <= MaxIDLength && shareSlugPattern.MatchString(raw)) {
		return nil
	}
	return ErrInvalidShareID
}

// ValidateName accepts 1 to 255 characters with no control character.
func ValidateName(raw string) error {
	n := utf8.RuneCountInString(raw)
	if n < 1 || n > MaxNameLength {
		return ErrInvalidName
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			return ErrInvalidName
		}
	}
	return nil
}

// MaxEmailLength is the most characters an e-mail address may hold (the path length of RFC 5321).
const MaxEmailLength = 254

// ValidateEmail accepts one address of at most MaxEmailLength characters, with no space or control character, one "@", a
// local part, and a host with a dot inside it. It is a shape check only: the server decides whether the address may be used.
func ValidateEmail(raw string) error {
	if raw == "" || utf8.RuneCountInString(raw) > MaxEmailLength {
		return ErrInvalidEmail
	}
	for _, r := range raw {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return ErrInvalidEmail
		}
	}
	local, host, ok := strings.Cut(raw, "@")
	if !ok || local == "" || strings.Contains(host, "@") || !strings.Contains(host, ".") ||
		strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateText accepts at most MaxTextBytes bytes. An empty text is allowed.
func ValidateText(raw string) error {
	if len(raw) > MaxTextBytes {
		return ErrInvalidText
	}
	return nil
}

// ValidateTime accepts an RFC 3339 time with a zone. A time without one is refused, so the server never guesses it.
func ValidateTime(raw string) error {
	if _, err := time.Parse(time.RFC3339, raw); err != nil {
		return ErrInvalidTime
	}
	return nil
}

// ValidateIndex accepts a fractional index of 1 to MaxIndexLength printable ASCII characters, from 0x20 to 0x7E (the
// idx notation of the inventory). A space is printable; a control byte and any non-ASCII character are refused.
func ValidateIndex(raw string) error {
	if raw == "" || len(raw) > MaxIndexLength {
		return ErrInvalidIndex
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] < 0x20 || raw[i] > 0x7e {
			return ErrInvalidIndex
		}
	}
	return nil
}

// colorPattern is the hex colour of the inventory's color notation: #RGB or #RRGGBB.
var colorPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// ValidateColor accepts a hex colour, #RGB or #RRGGBB, in either letter case.
func ValidateColor(raw string) error {
	if !colorPattern.MatchString(raw) {
		return ErrInvalidColor
	}
	return nil
}

// ValidateIcon accepts an icon of the shape the wiki takes: one emoji, a named icon, or an attachment UUID. The wiki holds
// the list of named icons, so Retriever checks only that the text is not blank, is at most MaxNameLength characters, and has
// no control character. The server decides whether the name exists.
func ValidateIcon(raw string) error {
	if strings.TrimSpace(raw) == "" || utf8.RuneCountInString(raw) > MaxNameLength {
		return ErrInvalidIcon
	}
	for _, r := range raw {
		if unicode.IsControl(r) {
			return ErrInvalidIcon
		}
	}
	return nil
}

// ValidateScore accepts a finite number from -1 to 1 inclusive, the score of a search (the inventory's num[-1..1]).
func ValidateScore(v float64) error {
	if math.IsNaN(v) || v < -1 || v > 1 {
		return ErrInvalidScore
	}
	return nil
}

// ValidateIDList accepts at most MaxIDList IDs, and each one must pass ValidateID.
func ValidateIDList(ids []string) error {
	if len(ids) > MaxIDList {
		return ErrInvalidIDList
	}
	for _, id := range ids {
		if ValidateID(id) != nil {
			return ErrInvalidIDList
		}
	}
	return nil
}

// ValidateURL accepts an absolute HTTPS URL with a DNS host name and no user information. An IP literal, a single-label
// host, localhost, and a port are refused, as NewEndpoint refuses them.
func ValidateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return ErrInvalidURL
	}
	if !validHostName(u.Hostname()) {
		return ErrInvalidURL
	}
	return nil
}

// NewSearchQuery returns raw when it holds at least one character that is not a space and no more than MaxQueryLength
// characters. It does not change the text.
func NewSearchQuery(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" || utf8.RuneCountInString(raw) > MaxQueryLength {
		return "", ErrInvalidQuery
	}
	return raw, nil
}

// CheckPage validates one page of a list operation. The offset is never negative, and the limit is from 1 to
// MaxPageLimit.
func CheckPage(offset, limit int) error {
	if offset < 0 {
		return ErrInvalidOffset
	}
	if limit < 1 || limit > MaxPageLimit {
		return ErrInvalidLimit
	}
	return nil
}
