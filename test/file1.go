package main

type Pill struct {
	Name string `arg:"required"`
	Color int64 `arg:"required"`
	Amount uint8 `arg:"min=1,max=100"`
}