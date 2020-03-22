package xorm

import (
	"reflect"
	"testing"

	"xorm.io/core"
)

func TestParseOracleConnStr(t *testing.T) {
	tests := []struct {
		in       string
		expected string
		valid    bool
	}{
		{"user/pass@tcp(server:1521)/db", "db", true},
		{"user/pass@server:1521/db", "db", true},
		// test for net service name : https://docs.oracle.com/cd/B13789_01/network.101/b10775/glossary.htm#i998113
		{"user/pass@server:1521", "", true},
		{"user/pass@", "", false},
		{"user/pass", "", false},
		{"", "", false},
	}
	driver := core.QueryDriver("oci8")
	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			driver := driver
			uri, err := driver.Parse("oci8", test.in)
			if err != nil && test.valid {
				t.Errorf("%q got unexpected error: %s", test.in, err)
			} else if err == nil && !reflect.DeepEqual(test.expected, uri.DbName) {
				t.Errorf("%q got: %#v want: %#v", test.in, uri.DbName, test.expected)
			}
		})
	}
}
