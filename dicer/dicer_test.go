package dicer

import (
	"testing"
)

type DicerTest struct {
	template string
	inputs   []string
	output   string
	err      string
}

func TestDicer(T *testing.T) {
	tests := []DicerTest{
		// Simple expansions
		DicerTest{
			template: "hello %1",
			inputs:   []string{"world"},
			output:   "hello world",
		},
		DicerTest{
			template: "hello %[1]",
			inputs:   []string{"world"},
			output:   "hello world",
		},
		DicerTest{
			template: "%1 %2",
			inputs:   []string{"hello", "world"},
			output:   "hello world",
		},

		// Multi-character index
		DicerTest{
			template: "%10",
			inputs:   []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			output:   "10",
		},
		DicerTest{
			template: "%[10]",
			inputs:   []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			output:   "10",
		},
		DicerTest{
			template: "%10test",
			inputs:   []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			output:   "10test",
		},

		// Out of bound index
		DicerTest{
			template: "hello %2",
			inputs:   []string{"a"},
			err:      "template references out of bounds input index 2",
		},

		// Missing ]
		DicerTest{
			template: "xhello %[1 world",
			inputs:   []string{"hello"},
			err:      "dicer expression: character 10: missing closing ]",
		},

		DicerTest{
			template: "hello %[",
			inputs:   []string{},
			err:      "dicer expression: character 9: missing closing ]",
		},

		// %% escape
		DicerTest{
			template: "test%%%1",
			inputs:   []string{"hello"},
			output:   "test%hello",
		},

		// EOL %
		DicerTest{
			template: "test%",
			inputs:   []string{},
			output:   "test%",
		},

		// % followed by a non-number
		DicerTest{
			template: "test%q",
			inputs:   []string{},
			output:   "test%q",
		},

		// simple dice
		DicerTest{
			template: "%[1/2]",
			inputs:   []string{"1/2/3"},
			output:   "2",
		},

		// multi-level dice
		DicerTest{
			template: "%[1/2,1]",
			inputs:   []string{"1,a/2,b/3,c"},
			output:   "2",
		},

		// dice position out of bounds
		DicerTest{
			template: "%[1/4]",
			inputs:   []string{"1/2/3"},
			output:   "",
		},
	}

	for _, test := range tests {
		d := NewTemplate(test.template)
		output, err := d.Expand(test.inputs)

		if err != nil && test.err == "" {
			T.Errorf("template %v with %v: got unexpected error %q", test.template, test.inputs, err.Error())
		} else if test.err != "" {
			if err == nil {
				T.Errorf("template %v with %v: expected error %q, did not get any error", test.template, test.inputs, test.err)
			} else if err.Error() != test.err {
				T.Errorf("template %v with %v: expected error %q, got %q", test.template, test.inputs, test.err, err.Error())
			}
		} else if output != test.output {
			T.Errorf("template %v with %v: expected %v, got %v", test.template, test.inputs, test.output, output)
		}
	}
}
