
# argumenter



# Usage

```
argumenter -type T1,T2 -out OUTPUT INPUT
```

# Input code

* file name is `file.go`

```go
package main
type MyType struct {
    I int `arg:"default=5,min=0,max=10"`
    S string `arg:"required"`
    SI []int `arg:"required,lenmin=1,lenmax=4"`
}
```

# Generate command

```
argumenter -type MyType file.go
```

# Generate code

* output file name is `file_argumenter.go`

```go
func (m *MyType) Valid() error {
    if m.I < 0 || m.I > 10 {
        return errors.New()
    }
    if m.I == 0 {
        m.I = 5
    }
    if m.S == "" {
        return errors.New()
    }
    if m.SI == nil || len(m.SI) < 1 || len(m.SI) > 4 {
        return errors.New()
    }
    return nil
}
```
