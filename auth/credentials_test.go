package auth

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseCredentials_valid(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Credentials
	}{
		{
			name: "valid credentials",
			content: `{
				"installed": {
					"client_id": "foobar",
					"client_secret": "supersecret"
				}
			}`,
			want: Credentials{
				ClientID:     "foobar",
				ClientSecret: "supersecret",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got Credentials
			err := json.Unmarshal([]byte(test.content), &got)
			if err != nil {
				t.Fatalf("error\nwant: %#v\ngot error: %q", test.want, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("wrong result\nwant: %#v\ngot:  %#v", test.want, got)
			}
		})
	}
}

func TestDecodeCredentials_invalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "empty",
			content: "",
			wantErr: "unexpected end of JSON input",
		},
		{
			name:    "invalid JSON",
			content: "{",
			wantErr: "unexpected end of JSON input",
		},
		{
			name: "missing client_id",
			content: `{
				"installed": {
					"client_secret": "supersecret"
				}
			}`,
			wantErr: "invalid credentials: missing 'installed.client_id'",
		},
		{
			name: "missing client_secret",
			content: `{
				"installed": {
					"client_id": "foobar"
				}
			}`,
			wantErr: "invalid credentials: missing 'installed.client_secret'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got Credentials
			err := json.Unmarshal([]byte(test.content), &got)
			if err == nil {
				t.Fatalf("wrong result\nwant err containing: %q\ngot: nil\nunexpected result: %#v", test.wantErr, got)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("wrong error\nwant err containing: %q\ngot: %q\nunexpected result: %#v", test.wantErr, err, got)
			}
		})
	}
}
