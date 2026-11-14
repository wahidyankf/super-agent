package outline

import (
	"testing"
)

// The creators of a share, a template, and a file operation are the objects the router sends (presenters/share.ts,
// template.ts, fileOperation.ts): createdBy, updatedBy, and user. A flat createdById or userId is not in those replies,
// so reading one would always give an empty value in production (finding F3).

const uuidCreator = "33333333-3333-4333-8333-333333333333"

// TestShareCreatorIsTheCreatedByObjectsID checks that a share's creator is the id of its createdBy object, and that a
// flat createdById the router never sends is not read.
func TestShareCreatorIsTheCreatedByObjectsID(t *testing.T) {
	rec, err := projectShareInfo(decodeJSON(t, `{"data":{"shares":[{"id":"`+uuidShare+
		`","published":true,"createdBy":{"id":"`+uuidCreator+`","name":"A"}}]}}`))
	if err != nil {
		t.Fatalf("projectShareInfo: %v", err)
	}
	if v, _ := rec.Get("createdById"); v != uuidCreator {
		t.Fatalf("createdById = %v, want the createdBy object's id %s", v, uuidCreator)
	}
	flat, err := projectShareInfo(decodeJSON(t, `{"data":{"shares":[{"id":"`+uuidShare+`","createdById":"`+uuidOther+`"}]}}`))
	if err != nil {
		t.Fatalf("projectShareInfo(flat): %v", err)
	}
	if v, _ := flat.Get("createdById"); v != nil {
		t.Fatalf("createdById = %v from a flat field the router never sends, want none", v)
	}
}

// TestTemplateCreatorsAreTheCreatedByAndUpdatedByIDs checks that a template's creator and last editor are the ids of
// its createdBy and updatedBy objects, which is what presenters/template.ts sends.
func TestTemplateCreatorsAreTheCreatedByAndUpdatedByIDs(t *testing.T) {
	rec, err := projectObjectOf(templateSpec)(decodeJSON(t, `{"data":{"id":"`+uuidShare+`","title":"Weekly",`+
		`"createdBy":{"id":"`+uuidCreator+`"},"updatedBy":{"id":"`+uuidOther+`"}}}`))
	if err != nil {
		t.Fatalf("templateSpec: %v", err)
	}
	if v, _ := rec.Get("createdById"); v != uuidCreator {
		t.Fatalf("createdById = %v, want %s", v, uuidCreator)
	}
	if v, _ := rec.Get("updatedById"); v != uuidOther {
		t.Fatalf("updatedById = %v, want %s", v, uuidOther)
	}
}

// TestFileOperationUserIsTheUserObjectsID checks that a file operation names its user by the id of the user object,
// as presenters/fileOperation.ts sends it, and not by a flat userId.
func TestFileOperationUserIsTheUserObjectsID(t *testing.T) {
	rec, err := projectObjectOf(fileOperationSpec)(decodeJSON(t, `{"data":{"id":"`+uuidShare+`","type":"export",`+
		`"state":"complete","user":{"id":"`+uuidCreator+`","name":"A"}}}`))
	if err != nil {
		t.Fatalf("fileOperationSpec: %v", err)
	}
	if v, _ := rec.Get("userId"); v != uuidCreator {
		t.Fatalf("userId = %v, want the user object's id %s", v, uuidCreator)
	}
}

// TestOAuthAuthenticationClientIsItsNameOnly checks the client of an OAuth sign-in. presenters/oauthClient.ts
// (presentPublishedOAuthClient) sends the client's name and public fields, and no id, so the projection keeps the name.
func TestOAuthAuthenticationClientIsItsNameOnly(t *testing.T) {
	obj, _ := decodeJSON(t, `{"id":"`+uuidShare+`","userId":"`+uuidCreator+`",`+
		`"oauthClient":{"name":"Editor","clientId":"c-1","id":"`+uuidOther+`"},"createdAt":"2026-01-01T00:00:00Z"}`).(map[string]any)
	rec, err := oauthAuthenticationSpec(obj)
	if err != nil {
		t.Fatalf("oauthAuthenticationSpec: %v", err)
	}
	client := rec.Rec("oauthClient")
	if got, _ := client.Get("name"); got != "Editor" {
		t.Fatalf("oauthClient.name = %v, want Editor", got)
	}
	if _, has := client.Get("id"); has {
		t.Fatal("oauthClient keeps an id the router does not send")
	}
}
