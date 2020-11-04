package main

import (
	"bytes"
	"fmt"
	"math"
	"text/template"

	"github.com/bspaans/bleep/instruments"
	"github.com/bspaans/jit-compiler/ir"
	"github.com/bspaans/jit-compiler/ir/expr"
	"github.com/bspaans/jit-compiler/ir/shared"
	"github.com/bspaans/jit-compiler/ir/statements"
)

func CompileGeneratorDef(instrDef *instruments.GeneratorDef) []shared.IR {
	return []shared.IR{}
}

func CreateSineWavTable(bitDepth, tableSize int) []int {
	if bitDepth == 8 {
		return Create8bitSineWavTable(tableSize)
	}
	panic("Bit depth not supported in jit compiler")
}

func Create8bitSineWavTable(tableSize int) []int {
	result := make([]int, tableSize)

	maxValue := 256.0
	angle := math.Pi * 2.0 / float64(tableSize)
	for i := 0; i < tableSize; i++ {
		v := math.Sin(float64(i) * angle)
		scaled := (v + 1) * (maxValue / 2)
		maxClipped := math.Max(0, math.Ceil(scaled))
		result[i] = int(math.Min(maxClipped, maxValue-1))
	}
	return result
}

func CompilePrelude(sampleRate, tableSize, generatorCount, nrOfSamples int) []shared.IR {

	// Generate the sine table
	sineTable := Create8bitSineWavTable(tableSize)
	sineTableIR := make([]shared.IRExpression, len(sineTable))
	for i, v := range sineTable {
		sineTableIR[i] = expr.NewIR_Uint8(uint8(v))
	}

	// Generate the phase table
	phaseTableIR := make([]shared.IRExpression, generatorCount)
	for i := 0; i < generatorCount; i++ {
		phaseTableIR[i] = expr.NewIR_Float64(0.0)
	}

	// Generate the output table
	outputIR := make([]shared.IRExpression, nrOfSamples)
	for i := 0; i < nrOfSamples; i++ {
		outputIR[i] = expr.NewIR_Uint8(0)
	}
	sineGenerator := template.Must(template.New("test").Parse(SineGenerator))
	sineCode := []byte{}
	w := bytes.NewBuffer(sineCode)
	err := sineGenerator.Execute(w, TemplateData{
		TableSizeOverSampleRate: float64(tableSize) / float64(sampleRate),
	})
	if err != nil {
		panic(err)
	}

	return []shared.IR{
		statements.NewIR_Assignment("sine", expr.NewIR_StaticArray(shared.TUint8, sineTableIR)),
		statements.NewIR_Assignment("phaseTable", expr.NewIR_StaticArray(shared.TFloat64, phaseTableIR)),
		statements.NewIR_Assignment("N", expr.NewIR_Uint64(uint64(nrOfSamples))),
		statements.NewIR_Assignment("freq", expr.NewIR_Float64(440.0)),
		statements.NewIR_Assignment("output", expr.NewIR_StaticArray(shared.TUint8, outputIR)),
		ir.MustParseIR(w.String()),
		statements.NewIR_Assignment("result", expr.NewIR_ArrayIndex(expr.NewIR_Variable("sine"), expr.NewIR_Uint64(1))),
		statements.NewIR_Return(expr.NewIR_Variable("result")),
	}
}

type TemplateData struct {
	TableSizeOverSampleRate float64
}

var SineGenerator = `
tableDelta = freq * {{.TableSizeOverSampleRate}};
currentIndex = 0;
i = 0;
while i != N {
  output[i] = sine[i] * tableDelta;
  i = i + 1
}
`

func main() {

	// parse: sine + transpose(7, sine) or read YAML => instrDef
	// compile to IR
	// add prelude
	// compile to machine code
	// implement callback that updates memory locations (nr of samples) and reads results

	prelude := CompilePrelude(44100, 12, 1, 32)
	fmt.Println(prelude)
	b, err := ir.Compile(prelude)
	if err != nil {
		panic(err)
	}
	fmt.Println(b)
	fmt.Println(b.Execute())
}
