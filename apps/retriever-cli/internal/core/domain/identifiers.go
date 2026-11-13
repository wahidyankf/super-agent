package domain

// CollectionID is a collection identifier that passed validation. Build it with NewCollectionID.
type CollectionID struct {
	value string
}

// DocumentID is a document identifier that passed validation. Build it with NewDocumentID.
type DocumentID struct {
	value string
}

// NewCollectionID validates raw against ^[A-Za-z0-9_-]+$.
func NewCollectionID(raw string) (CollectionID, error) {
	if ValidateID(raw) != nil {
		return CollectionID{}, ErrInvalidCollectionID
	}
	return CollectionID{value: raw}, nil
}

// NewDocumentID validates raw against ^[A-Za-z0-9_-]+$.
func NewDocumentID(raw string) (DocumentID, error) {
	if ValidateID(raw) != nil {
		return DocumentID{}, ErrInvalidDocumentID
	}
	return DocumentID{value: raw}, nil
}

// String returns the validated identifier.
func (id CollectionID) String() string {
	return id.value
}

// String returns the validated identifier.
func (id DocumentID) String() string {
	return id.value
}
