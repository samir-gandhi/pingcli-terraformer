package exporter

import (
    "testing"
    "github.com/samir-gandhi/pingcli-plugin-terraformer/internal/api"
)

func TestShouldSkipConnector(t *testing.T) {
    cases := []struct{
        name string
        summary api.ConnectorInstanceSummary
        want bool
    }{
        {name: "skip skUserPool exact", summary: api.ConnectorInstanceSummary{ConnectorID: "skUserPool"}, want: true},
        {name: "skip skUserPool case-insensitive", summary: api.ConnectorInstanceSummary{ConnectorID: "SkUserPool"}, want: true},
        {name: "do not skip other connector", summary: api.ConnectorInstanceSummary{ConnectorID: "httpConnector"}, want: false},
        {name: "do not skip empty id", summary: api.ConnectorInstanceSummary{ConnectorID: ""}, want: false},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := shouldSkipConnector(tc.summary)
            if got != tc.want {
                t.Fatalf("shouldSkipConnector(%v) = %v; want %v", tc.summary.ConnectorID, got, tc.want)
            }
        })
    }
}
