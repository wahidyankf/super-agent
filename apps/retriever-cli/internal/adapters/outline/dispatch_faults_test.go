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
	gw := clientFaultGateway(t, clientFaultRT(func(*http.Request) (*http.Response, error) {
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

// TestDispatchRefusesEachOperationOutsideItsTable checks the entry points. Each refuses an unknown operation, an
// operation of the wrong class, and an operation with no endpoint in its table, and in every case no request is sent.
func TestDispatchRefusesEachOperationOutsideItsTable(t *testing.T) {
	adminOp := requestCasesC[0]
	tests := []struct {
		name string
		call func(gw *Gateway) error
		want error
	}{
		{
			name: "Read refuses an unknown operation",
			call: func(gw *Gateway) error {
				_, err := gw.Read(context.Background(), "outline.nope", domain.NewArgs(nil))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Read refuses a read of another kind that has no Outline endpoint",
			call: func(gw *Gateway) error {
				_, err := gw.Read(context.Background(), domain.OpKiwiProductsList, domain.NewArgs(map[string]any{}))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Write refuses an unknown operation",
			call: func(gw *Gateway) error {
				_, err := gw.Write(context.Background(), "outline.nope", domain.NewArgs(nil))
				return err
			},
			want: domain.ErrUnsupportedOperation,
		},
		{
			name: "Write refuses a read operation",
			call: func(gw *Gateway) error {
				_, err := gw.Write(context.Background(), domain.OpCollectionsInfo, domain.NewArgs(map[string]any{}))
				return err
			},
			want: domain.ErrWrongOperationClass,
		},
		{
			name: "Write refuses a write with missing arguments",
			call: func(gw *Gateway) error {
				_, err := gw.Write(context.Background(), adminOp.op, domain.NewArgs(map[string]any{}))
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sent := 0
			gw := clientFaultGateway(t, clientFaultRT(func(*http.Request) (*http.Response, error) {
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
