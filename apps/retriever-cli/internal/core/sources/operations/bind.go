package operations

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// standardInputMarker is the whole value of a text parameter that asks to read standard input. Retriever does not read
// standard input, so the marker is refused.
const standardInputMarker = "-"

// issueKeyPattern is the shape of a Jira issue key (ADR 010 section 2): an upper-case project key of 2 to 20 characters,
// a dash, and a number with no leading zero of at most 9 digits. Retriever never corrects a key that does not match it.
var issueKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,19}-[1-9][0-9]{0,8}$`)

// SourceParam is the name of the argument that selects the source. It is on every operation, so Bind accepts it on
// every row, and the use case reads it before the binder runs.
const SourceParam = "source"

// Bind turns the raw arguments of one call into the validated Args of row. The CLI passes flag and positional text,
// and MCP passes the decoded JSON values; both go through here, so a value is checked the same way on every surface.
// A parameter the caller does not set takes its default. An argument the row does not declare is refused, and so is a
// missing required argument. Bind makes no call.
func Bind(row Row, raw map[string]any) (domain.Args, error) {
	for name := range raw {
		if name == SourceParam {
			continue
		}
		if _, ok := row.Param(name); !ok {
			return domain.Args{}, domain.ErrUnknownArgument
		}
	}
	values := map[string]any{}
	for _, p := range row.Params {
		v, ok := raw[p.Name]
		if !ok || v == nil {
			if p.Required {
				return domain.Args{}, domain.ErrMissingArgument
			}
			if p.Default == nil {
				continue
			}
			v = p.Default
		}
		out, err := convert(p, v)
		if err != nil {
			return domain.Args{}, err
		}
		values[p.Name] = out
	}
	if err := checkOneOf(row, values); err != nil {
		return domain.Args{}, err
	}
	if err := checkAnyOf(row, values); err != nil {
		return domain.Args{}, err
	}
	return domain.NewArgs(values), nil
}

// checkOneOf checks a row's exactly-one rule against the set values: one of the named parameters must be set, and no
// more than one. A row with no such rule passes.
func checkOneOf(row Row, values map[string]any) error {
	if len(row.OneOf) == 0 {
		return nil
	}
	set := 0
	for _, name := range row.OneOf {
		if _, ok := values[name]; ok {
			set++
		}
	}
	if set != 1 {
		return domain.ErrExactlyOneArgument
	}
	return nil
}

// checkAnyOf checks a row's at-least-one rule against the set values: one of the named parameters must be set. A row with
// no such rule passes, and a row whose named parameters are all unset is refused with ErrMissingArgument.
func checkAnyOf(row Row, values map[string]any) error {
	if len(row.AnyOf) == 0 {
		return nil
	}
	for _, name := range row.AnyOf {
		if _, ok := values[name]; ok {
			return nil
		}
	}
	return domain.ErrMissingArgument
}

// Validate checks the values of an Args again, with the rules of row. The use case and the Outline adapter both call
// it before a value goes into a request, so a value that reaches the wire has passed the catalog's rules.
func Validate(row Row, args domain.Args) error {
	_, err := Bind(row, args.Values())
	return err
}

// convert checks one value against its parameter and returns it in its Go type: string, int, bool, or []string.
func convert(p Param, v any) (any, error) {
	switch p.Kind {
	case KindID:
		s, err := asText(v, domain.ErrInvalidID)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateID(s)
	case KindCollectionID:
		s, err := asText(v, domain.ErrInvalidCollectionID)
		if err != nil {
			return nil, err
		}
		if _, err := domain.NewCollectionID(s); err != nil {
			return nil, err
		}
		return s, nil
	case KindDocumentID:
		s, err := asText(v, domain.ErrInvalidDocumentID)
		if err != nil {
			return nil, err
		}
		if _, err := domain.NewDocumentID(s); err != nil {
			return nil, err
		}
		return s, nil
	case KindIssueKey:
		s, err := asText(v, domain.ErrInvalidDocumentID)
		if err != nil {
			return nil, err
		}
		if !issueKeyPattern.MatchString(s) {
			return nil, domain.ErrInvalidDocumentID
		}
		return s, nil
	case KindCollectionUUID:
		s, err := asText(v, domain.ErrInvalidCollectionID)
		if err != nil {
			return nil, err
		}
		if domain.ValidateUUID(s) != nil {
			return nil, domain.ErrInvalidCollectionID
		}
		return s, nil
	case KindQuery:
		s, err := asText(v, domain.ErrInvalidQuery)
		if err != nil {
			return nil, err
		}
		return domain.NewSearchQuery(s)
	case KindLimit:
		n, err := asInt(v, domain.ErrInvalidLimit)
		if err != nil {
			return nil, err
		}
		if n < 1 || n > domain.MaxPageLimit {
			return nil, domain.ErrInvalidLimit
		}
		return n, nil
	case KindOffset:
		n, err := asInt(v, domain.ErrInvalidOffset)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, domain.ErrInvalidOffset
		}
		return n, nil
	case KindInt:
		n, err := asInt(v, domain.ErrInvalidInt)
		if err != nil {
			return nil, err
		}
		if n < p.Min || n > p.Max {
			// The message names the parameter, so a refused --limit says "limit", as every other refusal of a number does.
			return nil, fmt.Errorf("%s: %w", p.Name, domain.ErrInvalidInt)
		}
		return n, nil
	case KindBool:
		return asBool(v)
	case KindName:
		s, err := asText(v, domain.ErrInvalidName)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateName(s)
	case KindText:
		s, err := asText(v, domain.ErrInvalidText)
		if err != nil {
			return nil, err
		}
		// Retriever reads no standard input, so a dash is refused rather than sent as the literal text (finding B1).
		if s == standardInputMarker {
			return nil, domain.ErrStandardInputUnsupported
		}
		return s, domain.ValidateText(s)
	case KindReason:
		s, err := asText(v, domain.ErrInvalidReason)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateReason(s)
	case KindEnum:
		s, err := asText(v, domain.ErrInvalidEnum)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(p.Values, s) {
			return nil, domain.ErrInvalidEnum
		}
		return s, nil
	case KindIDList:
		ids, err := asIDs(v)
		if err != nil {
			return nil, err
		}
		return ids, domain.ValidateIDList(ids)
	case KindEventList:
		names, err := asStrings(v, domain.ErrInvalidName)
		if err != nil {
			return nil, err
		}
		return names, domain.ValidateEventList(names)
	case KindURLList:
		urls, err := asStrings(v, domain.ErrInvalidURL)
		if err != nil {
			return nil, err
		}
		return urls, domain.ValidateURLList(urls)
	case KindUUID:
		s, err := asText(v, domain.ErrInvalidUUID)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateUUID(s)
	case KindShareID:
		s, err := asText(v, domain.ErrInvalidShareID)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateShareID(s)
	case KindUUIDList:
		ids, err := asStrings(v, domain.ErrInvalidUUID)
		if err != nil {
			return nil, err
		}
		if len(ids) > domain.MaxIDList {
			return nil, domain.ErrInvalidUUID
		}
		for _, id := range ids {
			if err := domain.ValidateUUID(id); err != nil {
				return nil, err
			}
		}
		return ids, nil
	case KindEnumList:
		items, err := asStrings(v, domain.ErrInvalidEnum)
		if err != nil {
			return nil, err
		}
		// A list holds each value at most once, so it is never longer than the values it may pick from. A repeated value
		// is sent once, in its first position.
		if len(items) > len(p.Values) {
			return nil, domain.ErrInvalidEnum
		}
		out := make([]string, 0, len(items))
		for _, item := range items {
			if !slices.Contains(p.Values, item) {
				return nil, domain.ErrInvalidEnum
			}
			if !slices.Contains(out, item) {
				out = append(out, item)
			}
		}
		return out, nil
	case KindTime:
		s, err := asText(v, domain.ErrInvalidTime)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateTime(s)
	case KindURL:
		s, err := asText(v, domain.ErrInvalidURL)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateURL(s)
	case KindIndex:
		s, err := asText(v, domain.ErrInvalidIndex)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateIndex(s)
	case KindColor:
		s, err := asText(v, domain.ErrInvalidColor)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateColor(s)
	case KindEmail:
		s, err := asText(v, domain.ErrInvalidEmail)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateEmail(s)
	case KindIcon:
		s, err := asText(v, domain.ErrInvalidIcon)
		if err != nil {
			return nil, err
		}
		return s, domain.ValidateIcon(s)
	case KindScore:
		n, err := asNumber(v, domain.ErrInvalidScore)
		if err != nil {
			return nil, err
		}
		return n, domain.ValidateScore(n)
	}
	return nil, domain.ErrUnsupportedOperation
}

// asText returns v as text. Any other type is the bad error.
func asText(v any, bad error) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", bad
	}
	return s, nil
}

// asInt returns v as an integer. It accepts an int, a whole JSON number, or a decimal text, and any other value is the
// bad error.
func asInt(v any, bad error) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case float64:
		if n != math.Trunc(n) || n > math.MaxInt32 || n < math.MinInt32 {
			return 0, bad
		}
		return int(n), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil {
			return 0, bad
		}
		return parsed, nil
	}
	return 0, bad
}

// asNumber returns v as a number. It accepts a float or an int, or the decimal text of one, as the command line gives it.
// Any other value is the bad error. A NaN or an infinity is returned as it parses, and the validator refuses it.
func asNumber(v any, bad error) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return 0, bad
		}
		return parsed, nil
	}
	return 0, bad
}

// asBool returns v as a boolean. It accepts a bool or the text true or false.
func asBool(v any) (bool, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		switch b {
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
	}
	return false, domain.ErrInvalidBool
}

// asIDs returns v as a list of IDs. Any other value is ErrInvalidIDList.
func asIDs(v any) ([]string, error) {
	return asStrings(v, domain.ErrInvalidIDList)
}

// asStrings returns v as a list of text: a list of text, or one text of comma-separated items with the spaces around
// each item removed. Any other value, or a list with an item that is not text, is the bad error.
func asStrings(v any, bad error) ([]string, error) {
	switch list := v.(type) {
	case []string:
		return list, nil
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				return nil, bad
			}
			out = append(out, s)
		}
		return out, nil
	case string:
		parts := strings.Split(list, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			out = append(out, strings.TrimSpace(part))
		}
		return out, nil
	}
	return nil, bad
}
