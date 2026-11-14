package domain

// MaxURLList is the most redirect addresses an OAuth client may name (the field reference, oauthClients.create).
const MaxURLList = 10

// MaxURLLength is the longest address, in characters, that the wiki accepts for a redirect (the field reference).
const MaxURLLength = 1024

// ValidateURLList accepts between one and MaxURLList addresses, and each one must pass ValidateURL and hold at most
// MaxURLLength characters. A refused list is ErrInvalidURL, the same refusal as one address.
func ValidateURLList(urls []string) error {
	if len(urls) == 0 || len(urls) > MaxURLList {
		return ErrInvalidURL
	}
	for _, raw := range urls {
		if len(raw) > MaxURLLength {
			return ErrInvalidURL
		}
		if err := ValidateURL(raw); err != nil {
			return err
		}
	}
	return nil
}
