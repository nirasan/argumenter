package main

type MyType struct {
	I  int          `arg:"default=5,min=0,max=10"`
	S  string       `arg:"required"`
	SI []int        `arg:"required,lenmin=1,lenmax=4"`
	M  map[int]bool `arg:"required"`
	F  func()       `arg:"required"`
	IN interface{}  `arg:"required"`
	P  *int         `arg:"required"`
}
