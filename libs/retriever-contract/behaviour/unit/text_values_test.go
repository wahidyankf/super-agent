package unit_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/wahidyankf/super-agent/libs/retriever-contract/domain"
)

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A name of 1 to 255 characters with no control character is accepted
func TestTextValuesAcceptNamesOfOneToTwoHundredFiftyFiveCharacters(t *testing.T) {
	for _, raw := range []string{"Engineering", "R&D = Drafts", "Ünïcode name", "x"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateName(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An empty name is refused
func TestTextValuesRefuseEmptyName(t *testing.T) {
	wantSentinel(t, domain.ValidateName(""), domain.ErrInvalidName)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A name of 256 characters is refused
func TestTextValuesRefuseNameOf256Characters(t *testing.T) {
	wantSentinel(t, domain.ValidateName(strings.Repeat("a", 256)), domain.ErrInvalidName)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A name with a tab character is refused
func TestTextValuesRefuseNameWithTabCharacter(t *testing.T) {
	wantSentinel(t, domain.ValidateName("two\twords"), domain.ErrInvalidName)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An empty text is accepted
func TestTextValuesAcceptEmptyText(t *testing.T) {
	noError(t, domain.ValidateText(""))
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A text of exactly 1 MiB is accepted
func TestTextValuesAcceptTextOfExactlyOneMiB(t *testing.T) {
	noError(t, domain.ValidateText(strings.Repeat("a", 1048576)))
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A text one byte longer than 1 MiB is refused
func TestTextValuesRefuseTextOneByteOverOneMiB(t *testing.T) {
	wantText(t, domain.ValidateText(strings.Repeat("a", 1048577)), "text must be at most 1 MiB")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A reason of up to 2000 characters with no control character is accepted
func TestTextValuesAcceptReasonOfUpTo2000Characters(t *testing.T) {
	for _, length := range []int{0, 1, 2000} {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			noError(t, domain.ValidateReason(strings.Repeat("a", length)))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A reason of 2001 characters is refused
func TestTextValuesRefuseReasonOf2001Characters(t *testing.T) {
	wantText(t, domain.ValidateReason(strings.Repeat("a", 2001)),
		"invalid reason: use at most 2000 characters with no control character")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A reason with a line feed character is refused
func TestTextValuesRefuseReasonWithLineFeed(t *testing.T) {
	wantText(t, domain.ValidateReason("two\nwords"),
		"invalid reason: use at most 2000 characters with no control character")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An e-mail address with a local part and a dotted host is accepted
func TestTextValuesAcceptEmailWithLocalPartAndDottedHost(t *testing.T) {
	for _, raw := range []string{"name@example.com", "a.b@sub.example.test"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateEmail(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An e-mail address without a local part or a dotted host is refused
func TestTextValuesRefuseEmailWithoutLocalPartOrDottedHost(t *testing.T) {
	for _, raw := range []string{
		"name@",
		"@example.com",
		"name@example",
		"two@@example.com",
		"a b@example.com",
		"name@.example.com",
		"",
	} {
		t.Run(raw, func(t *testing.T) {
			wantSentinel(t, domain.ValidateEmail(raw), domain.ErrInvalidEmail)
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A time in RFC 3339 with a zone is accepted
func TestTextValuesAcceptRFC3339TimeWithZone(t *testing.T) {
	for _, raw := range []string{"2026-11-21T10:00:00Z", "2026-11-21T10:00:00+07:00"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateTime(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A time without a zone, or in another format, is refused
func TestTextValuesRefuseTimeWithoutZoneOrInAnotherFormat(t *testing.T) {
	for _, raw := range []string{"2026-11-21T10:00:00", "2026-11-21", "21/11/2026", ""} {
		t.Run(raw, func(t *testing.T) {
			wantText(t, domain.ValidateTime(raw), "invalid time: use RFC 3339 with a zone")
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A fractional index of 1 to 256 printable ASCII characters is accepted
func TestTextValuesAcceptFractionalIndexOfPrintableASCII(t *testing.T) {
	for _, raw := range []string{"a0", "1", "a b", "~"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateIndex(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A fractional index that is empty or holds a non-ASCII character is refused
func TestTextValuesRefuseFractionalIndexEmptyOrNonASCII(t *testing.T) {
	for _, raw := range []string{"", "é"} {
		t.Run(raw, func(t *testing.T) {
			wantText(t, domain.ValidateIndex(raw), "invalid index: use 1 to 256 printable ASCII characters")
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A fractional index with a tab character is refused
func TestTextValuesRefuseFractionalIndexWithTabCharacter(t *testing.T) {
	wantText(t, domain.ValidateIndex("a\tb"), "invalid index: use 1 to 256 printable ASCII characters")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A fractional index of 257 characters is refused
func TestTextValuesRefuseFractionalIndexOf257Characters(t *testing.T) {
	wantText(t, domain.ValidateIndex(strings.Repeat("a", 257)), "invalid index: use 1 to 256 printable ASCII characters")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A hex colour of three or six digits is accepted in either letter case
func TestTextValuesAcceptHexColourOfThreeOrSixDigits(t *testing.T) {
	for _, raw := range []string{"#3a7bd5", "#3ad", "#ABCDEF"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateColor(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A colour without a hash, with another digit count, or a name is refused
func TestTextValuesRefuseColourWithoutHashOtherDigitCountOrName(t *testing.T) {
	for _, raw := range []string{"3a7bd5", "#3a7bd", "#3a7bd5ff", "red"} {
		t.Run(raw, func(t *testing.T) {
			wantText(t, domain.ValidateColor(raw), "invalid color: use a hex color such as #3a7bd5 or #3ad")
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A non-blank icon of at most 255 characters is accepted
func TestTextValuesAcceptNonBlankIconOfAtMost255Characters(t *testing.T) {
	for _, raw := range []string{"star", "\U0001F4C4", "Idea"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateIcon(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An empty icon is refused
func TestTextValuesRefuseEmptyIcon(t *testing.T) {
	wantText(t, domain.ValidateIcon(""), "invalid icon: use 1 to 255 characters with no control character")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An icon of only spaces is refused
func TestTextValuesRefuseIconOfOnlySpaces(t *testing.T) {
	wantText(t, domain.ValidateIcon("   "), "invalid icon: use 1 to 255 characters with no control character")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An icon of 256 characters is refused
func TestTextValuesRefuseIconOf256Characters(t *testing.T) {
	wantText(t, domain.ValidateIcon(strings.Repeat("a", 256)),
		"invalid icon: use 1 to 255 characters with no control character")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An icon with a tab character is refused
func TestTextValuesRefuseIconWithTabCharacter(t *testing.T) {
	wantText(t, domain.ValidateIcon("two\twords"), "invalid icon: use 1 to 255 characters with no control character")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A search score from -1 to 1 inclusive is accepted
func TestTextValuesAcceptSearchScoreFromMinusOneToOne(t *testing.T) {
	for _, score := range []float64{-1, 0, 0.5, 1} {
		t.Run(fmt.Sprint(score), func(t *testing.T) {
			noError(t, domain.ValidateScore(score))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A search score outside -1 to 1, or not a number, is refused
func TestTextValuesRefuseSearchScoreOutsideRangeOrNotANumber(t *testing.T) {
	for _, score := range []float64{1.5, -1.01, math.NaN(), math.Inf(1)} {
		t.Run(fmt.Sprint(score), func(t *testing.T) {
			wantText(t, domain.ValidateScore(score), "invalid score: use a number from -1 to 1")
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A list of 50 valid IDs is accepted
func TestTextValuesAcceptListOf50ValidIDs(t *testing.T) {
	var ids []string
	for i := 1; i <= 50; i++ {
		ids = append(ids, fmt.Sprintf("id-%02d", i))
	}
	noError(t, domain.ValidateIDList(ids))
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A list of 51 IDs is refused
func TestTextValuesRefuseListOf51IDs(t *testing.T) {
	var ids []string
	for i := 1; i <= 51; i++ {
		ids = append(ids, fmt.Sprintf("id-%02d", i))
	}
	wantText(t, domain.ValidateIDList(ids), "invalid ID list: at most 50 valid IDs")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A list with an ID that breaks the ID grammar is refused
func TestTextValuesRefuseListWithIDThatBreaksTheGrammar(t *testing.T) {
	wantText(t, domain.ValidateIDList([]string{"doc-1", "doc 2"}), "invalid ID list: at most 50 valid IDs")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An event name of an asterisk or a dotted pair of letters is accepted
func TestTextValuesAcceptEventNameOfAsteriskOrDottedPair(t *testing.T) {
	for _, event := range []string{"*", "documents.create", "Documents.Create"} {
		t.Run(event, func(t *testing.T) {
			noError(t, domain.ValidateEventList([]string{event}))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An event name that is not an asterisk or a dotted pair is refused
func TestTextValuesRefuseEventNameThatIsNotAsteriskOrDottedPair(t *testing.T) {
	for _, event := range []string{"documents", "documents.", "*.create", "documents.a.b"} {
		t.Run(event, func(t *testing.T) {
			wantSentinel(t, domain.ValidateEventList([]string{event}), domain.ErrInvalidName)
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A list of 51 event names is refused
func TestTextValuesRefuseListOf51EventNames(t *testing.T) {
	names := make([]string, 51)
	for i := range names {
		names[i] = "documents.create"
	}
	wantSentinel(t, domain.ValidateEventList(names), domain.ErrInvalidName)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An HTTPS address with a DNS host name is accepted
func TestTextValuesAcceptHTTPSAddressWithDNSHostName(t *testing.T) {
	for _, raw := range []string{"https://example.com/x", "https://docs.example.test/a"} {
		t.Run(raw, func(t *testing.T) {
			noError(t, domain.ValidateURL(raw))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A URL that is not HTTPS, names no DNS host, or has credentials or a port
func TestTextValuesRefuseURLThatIsNotHTTPSOrHasNoDNSHost(t *testing.T) {
	for _, raw := range []string{
		"http://example.com",
		"https://localhost",
		"https://192.0.2.10",
		"https://user@example.com",
		"https://example.com:8443",
		"https://example",
		"https://Example.com",
	} {
		t.Run(raw, func(t *testing.T) {
			wantSentinel(t, domain.ValidateURL(raw), domain.ErrInvalidURL)
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A list of ten HTTPS addresses is accepted
func TestTextValuesAcceptListOfTenHTTPSAddresses(t *testing.T) {
	var urls []string
	for i := 1; i <= 10; i++ {
		urls = append(urls, fmt.Sprintf("https://example.com/r%d", i))
	}
	noError(t, domain.ValidateURLList(urls))
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An empty address list is refused
func TestTextValuesRefuseEmptyAddressList(t *testing.T) {
	wantSentinel(t, domain.ValidateURLList(nil), domain.ErrInvalidURL)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An address list of eleven valid HTTPS addresses is refused
func TestTextValuesRefuseAddressListOfElevenAddresses(t *testing.T) {
	var urls []string
	for i := 1; i <= 11; i++ {
		urls = append(urls, fmt.Sprintf("https://example.com/r%d", i))
	}
	wantSentinel(t, domain.ValidateURLList(urls), domain.ErrInvalidURL)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An address list holding one plain HTTP address is refused
func TestTextValuesRefuseAddressListHoldingPlainHTTPAddress(t *testing.T) {
	wantSentinel(t, domain.ValidateURLList([]string{"https://example.com/r1", "http://example.com/r2"}),
		domain.ErrInvalidURL)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An address list holding an address of 1025 characters is refused
func TestTextValuesRefuseAddressListHoldingAddressOf1025Characters(t *testing.T) {
	const prefix = "https://example.com/"
	addr := prefix + strings.Repeat("a", 1025-len(prefix))
	if len(addr) != 1025 {
		t.Fatalf("the address holds %d characters, want 1025", len(addr))
	}
	wantSentinel(t, domain.ValidateURLList([]string{addr}), domain.ErrInvalidURL)
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A search query with a non-space character is returned unchanged
func TestTextValuesReturnSearchQueryWithNonSpaceUnchanged(t *testing.T) {
	for _, raw := range []string{"install", "a b"} {
		t.Run(raw, func(t *testing.T) {
			got, err := domain.NewSearchQuery(raw)
			noError(t, err)
			if got != raw {
				t.Errorf("query = %q, want %q", got, raw)
			}
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A search query with spaces around a word is returned with the spaces
func TestTextValuesReturnSearchQueryWithSpacesAroundWord(t *testing.T) {
	const raw = "  term with spaces around"
	got, err := domain.NewSearchQuery(raw)
	noError(t, err)
	if got != raw {
		t.Errorf("query = %q, want %q with the spaces", got, raw)
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: An empty search query is refused
func TestTextValuesRefuseEmptySearchQuery(t *testing.T) {
	_, err := domain.NewSearchQuery("")
	wantText(t, err, "invalid query: at least one character and at most 1000 characters")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A search query of only spaces is refused
func TestTextValuesRefuseSearchQueryOfOnlySpaces(t *testing.T) {
	_, err := domain.NewSearchQuery("   ")
	wantText(t, err, "invalid query: at least one character and at most 1000 characters")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A search query of 1001 characters is refused
func TestTextValuesRefuseSearchQueryOf1001Characters(t *testing.T) {
	_, err := domain.NewSearchQuery(strings.Repeat("a", 1001))
	wantText(t, err, "invalid query: at least one character and at most 1000 characters")
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A page with an offset of zero or more and a limit of 1 to 100 is accepted
func TestTextValuesAcceptPageWithOffsetAndLimitInRange(t *testing.T) {
	cases := []struct{ offset, limit int }{{0, 1}, {40, 100}, {0, 25}}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("offset %d limit %d", tc.offset, tc.limit), func(t *testing.T) {
			noError(t, domain.CheckPage(tc.offset, tc.limit))
		})
	}
}

// @scenario specs/libs/retriever-contract/behaviours/text-values.feature: A page with a negative offset or a limit outside 1 to 100 is refused
func TestTextValuesRefusePageWithNegativeOffsetOrLimitOutsideRange(t *testing.T) {
	cases := []struct {
		offset, limit int
		message       string
	}{
		{-1, 25, "offset must not be negative"},
		{0, 0, "limit must be from 1 to 100"},
		{0, 101, "limit must be from 1 to 100"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("offset %d limit %d", tc.offset, tc.limit), func(t *testing.T) {
			wantText(t, domain.CheckPage(tc.offset, tc.limit), tc.message)
		})
	}
}
