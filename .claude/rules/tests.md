---
paths: ["**/*_test.go"]
---

# Go Test Conventions

## Structure

Use table-driven tests:

    func TestThing(t *testing.T) {
        tests := []struct {
            name    string
            input   string
            want    string
            wantErr bool
        }{
            {name: "happy path", input: "foo", want: "FOO"},
            {name: "empty input", input: "", wantErr: true},
        }
        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                got, err := Thing(tt.input)
                if (err != nil) != tt.wantErr {
                    t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
                }
                if got != tt.want {
                    t.Errorf("got %q, want %q", got, tt.want)
                }
            })
        }
    }

## Naming

- Unit test file: `foo_test.go`
- Integration test file: `foo_integration_test.go` with `//go:build integration` at top
- Test function: `TestFoo_Scenario` for specific scenarios, `TestFoo` for the main case

## Assertions

No third-party assertion libraries (`testify`, etc). Use standard library: `t.Errorf`, `t.Fatalf`.