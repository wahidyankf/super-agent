package outline

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/wahidyankf/super-agent/apps/retriever-cli/internal/core/domain"
)

// TestCurrentReadRefusesAnOperationWithNoEndpoint checks readCurrent: an operation with no endpoint in the read table is
// unsupported, and no request is sent.
func TestCurrentReadRefusesAnOperationWithNoEndpoint(t *testing.T) {
	sent := 0
	gw := clientFaultGateway(t, domain.AccessRead, clientFaultRT(func(*http.Request) (*http.Response, error) {
		sent++
		return clientFaultReply(http.StatusOK, `{"data":{}}`), nil
	}))
	reply := func(any) (domain.Record, error) { return domain.Record{}, nil }
	if _, err := gw.readCurrent(context.Background(), "outline.nope", domain.NewArgs(nil), reply); !errors.Is(err, domain.ErrUnsupportedOperation) {
		t.Fatalf("readCurrent error = %v, want ErrUnsupportedOperation", err)
	}
	if sent != 0 {
		t.Fatalf("readCurrent sent %d requests for an unsupported operation", sent)
	}
}

// TestDispatchRefusesEachOperationOutsideItsTable checks the three entry points. Each refuses an unknown operation, an
// operation of the wrong class, and an operation with no endpoint in its table, and in every case no request is sent.
func TestDispatchRefusesEachOperationOutsideItsTable(t *testing.T) {
	writeOp := requestCasesB[0]
	adminOp := requestCasesC[0]
	tests := []struct {
		name   string
		access domain.Access
		call   func(gw *Gateway) error
		want   error
	}{
		{
			name: "Read refuses an unknown operation", access: domain.AccessRead,
			call: func(gw *Gateway) error {
				_, err := gw.Read(context.Background(), "outline.nope", domain.NewArgs(nil))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Read refuses a read of another kind that has no Outline endpoint", access: domain.AccessRead,
			call: func(gw *Gateway) error {
				_, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Write refuses an operation with no write endpoint", access: domain.AccessAdmin,
			call: func(gw *Gateway) error {
				_, err := gw.Write(context.Background(), adminOp.op, domain.NewArgs(adminOp.args))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Write refuses an unknown operation", access: domain.AccessWrite,
			call: func(gw *Gateway) error {
				_, err := gw.Write(context.Background(), "outline.nope", domain.NewArgs(nil))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Administer refuses an unknown operation", access: domain.AccessAdmin,
			call: func(gw *Gateway) error {
				_, err := gw.Administer(context.Background(), "outline.nope", domain.NewArgs(nil))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Administer refuses a read operation", access: domain.AccessAdmin,
			call: func(gw *Gateway) error {
				_, err := gw.Administer(context.Background(), domain.OpCollectionsInfo, domain.NewArgs(map[string]any{}))
				return err
			},
			want: domain.ErrWrongOperationClass,
		},
		{
			name: "Administer refuses a write with missing arguments", access: domain.AccessAdmin,
			call: func(gw *Gateway) error {
				_, err := gw.Administer(context.Background(), adminOp.op, domain.NewArgs(map[string]any{}))
				return err
			},
		},
		{
			name: "Administer refuses a write that is not an admin operation", access: domain.AccessAdmin,
			call: func(gw *Gateway) error {
				_, err := gw.Administer(context.Background(), writeOp.op, domain.NewArgs(writeOp.args))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sent := 0
			gw := clientFaultGateway(t, tt.access, clientFaultRT(func(*http.Request) (*http.Response, error) {
				sent++
				return clientFaultReply(http.StatusOK, `{"data":{}}`), nil
			}))
			err := tt.call(gw)
			if err == nil {
				t.Fatalf("the call returned no error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if sent != 0 {
				t.Fatalf("the refused call sent %d requests", sent)
			}
		})
	}
}
