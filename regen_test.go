/*
Copyright 2014 Zachary Klippenstein

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package regen

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"regexp/syntax"
	"strings"
	"testing"
)

const (
	// Each expression is generated and validated this many times.
	SampleSize = 999

	// Arbitrary limit in the standard package.
	// See https://golang.org/src/regexp/syntax/parse.go?s=18885:18935#L796
	MaxSupportedRepeatCount = 1000
)

func ExampleGenerate() {
	pattern := "[ab]{5}"
	str, _ := Generate(pattern)

	if matched, _ := regexp.MatchString(pattern, str); matched {
		fmt.Println("Matches!")
	}

	// Output:
	// Matches!
}

func ExampleNewGenerator() {
	pattern := "[ab]{5}"

	// Note that this uses a constant seed, so the generated string
	// will always be the same across different runs of the program.
	// Use a more random seed for real use (e.g. time-based).
	generator, _ := NewGenerator(pattern, &GeneratorArgs{
		RngSource: rand.NewSource(0),
	})

	str := generator.Generate()

	if matched, _ := regexp.MatchString(pattern, str); matched {
		fmt.Println("Matches!")
	}

	// Output:
	// Matches!
}

func ExampleNewGenerator_perl() {
	pattern := `\d{5}`

	generator, _ := NewGenerator(pattern, &GeneratorArgs{
		Flags: syntax.Perl,
	})

	str := generator.Generate()

	if matched, _ := regexp.MatchString("[[:digit:]]{5}", str); matched {
		fmt.Println("Matches!")
	}
	// Output:
	// Matches!
}

func ExampleCaptureGroupHandler() {
	pattern := `Hello, (?P<firstname>[A-Z][a-z]{2,10}) (?P<lastname>[A-Z][a-z]{2,10})`

	generator, _ := NewGenerator(pattern, &GeneratorArgs{
		Flags: syntax.Perl,
		CaptureGroupHandler: func(index int, name string, group *syntax.Regexp, generator Generator, args *GeneratorArgs) string {
			if name == "firstname" {
				return fmt.Sprintf("FirstName (e.g. %s)", generator.Generate())
			}
			return fmt.Sprintf("LastName (e.g. %s)", generator.Generate())
		},
	})

	// Print to stderr since we're generating random output and can't assert equality.
	fmt.Fprintln(os.Stderr, generator.Generate())

	// Needed for "go test" to run this example. (Must be a blank line before.)

	// Output:
}

func TestGeneratorArgs(t *testing.T) {
	t.Parallel()

	t.Run("initialize", func(t *testing.T) {
		t.Parallel()

		t.Run("handles empty struct", func(t *testing.T) {
			t.Parallel()

			args := GeneratorArgs{}

			err := args.initialize()
			if err != nil {
				t.Fatalf("err should be nil")
			}
		})

		t.Run("Unicode groups not supported", func(t *testing.T) {
			t.Parallel()

			args := &GeneratorArgs{
				Flags: syntax.UnicodeGroups,
			}

			err := args.initialize()
			if err.Error() != "UnicodeGroups not supported" {
				t.Fatal("should be UnicodeGroups not supported")
			}
		})

		t.Run("Panics if repeat bounds are invalid", func(t *testing.T) {
			t.Parallel()

			args := &GeneratorArgs{
				MinUnboundedRepeatCount: 2,
				MaxUnboundedRepeatCount: 1,
			}

			defer func() {
				if r := recover(); r == nil {
					t.Fatal("The code did not panic")
				}
			}()

			args.initialize()
		})

		t.Run("Allows equal repeat bounds", func(t *testing.T) {
			t.Parallel()

			args := &GeneratorArgs{
				MinUnboundedRepeatCount: 1,
				MaxUnboundedRepeatCount: 1,
			}

			err := args.initialize()
			if err != nil {
				t.Fatalf("err should be nil")
			}
		})
	})

	t.Run("Rng", func(t *testing.T) {
		t.Parallel()

		t.Run("Panics if called before initialization", func(t *testing.T) {
			t.Parallel()

			args := GeneratorArgs{}

			defer func() {
				if r := recover(); r == nil {
					t.Fatal("The code did not panic")
				}
			}()

			args.Rng()
		})

		t.Run("Non-nil after initialization", func(t *testing.T) {
			t.Parallel()

			args := GeneratorArgs{}
			err := args.initialize()
			if err != nil {
				t.Fatalf("err should be nil")
			}
		})
	})
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	t.Run("Handles nil GeneratorArgs", func(t *testing.T) {
		t.Parallel()

		generator, err := NewGenerator("", nil)
		if err != nil {
			t.Fatalf("err should be nil")
		}
		if generator == nil {
			t.Fatalf("generator should not be nil")
		}
	})

	t.Run("Handles empty GeneratorArgs", func(t *testing.T) {
		t.Parallel()

		generator, err := NewGenerator("", &GeneratorArgs{})
		if err != nil {
			t.Fatalf("err should be nil")
		}
		if generator == nil {
			t.Fatalf("generator should not be nil")
		}
	})

	t.Run("Forwards errors from args initialization", func(t *testing.T) {
		t.Parallel()

		args := &GeneratorArgs{
			Flags: syntax.UnicodeGroups,
		}

		_, err := NewGenerator("", args)
		if err == nil {
			t.Fatalf("err should not be nil")
		}
	})
}

func TestGenEmpty(t *testing.T) {
	t.Parallel()

	args := &GeneratorArgs{
		RngSource: rand.NewSource(0),
	}

	GeneratesStringMatching(t, args, "", "^$")
}

func TestGenLiterals(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil,
		"a",
		"abc",
	)
}

func TestGenDotNotNl(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil, ".")

	generator, _ := NewGenerator(".", nil)

	// Not a very strong assertion, but not sure how to do better. Exploring the entire
	// generation space (2^32) takes far too long for a unit test.
	for i := 0; i < SampleSize; i++ {
		s := generator.Generate()
		if strings.Contains(s, "\n") {
			t.Fatalf("should not contain newline")
		}
	}
}

func TestGenStringStartEnd(t *testing.T) {
	t.Parallel()

	args := &GeneratorArgs{
		RngSource: rand.NewSource(0),
		Flags:     0,
	}

	GeneratesStringMatching(t, args, `^abc$`, `^abc$`)
	GeneratesStringMatching(t, args, `$abc^`, `^abc$`)
	GeneratesStringMatching(t, args, `a^b$c`, `^abc$`)
}

func TestGenQuestionMark(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil,
		"a?",
		"(abc)?",
		"[ab]?",
		".?")
}

func TestGenPlus(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil, "a+")
}

func TestGenStar(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil, "a*")

	t.Run("HitsDefaultMin", func(t *testing.T) {
		t.Parallel()

		regexp := "a*"
		args := &GeneratorArgs{
			RngSource: rand.NewSource(0),
		}
		counts := generateLenHistogram(regexp, DefaultMaxUnboundedRepeatCount, args)

		if counts[0] <= 0 {
			t.Fatalf("should be greater than 0")
		}
	})

	t.Run("HitsCustomMin", func(t *testing.T) {
		t.Parallel()

		regexp := "a*"
		args := &GeneratorArgs{
			RngSource:               rand.NewSource(0),
			MinUnboundedRepeatCount: 200,
		}
		counts := generateLenHistogram(regexp, DefaultMaxUnboundedRepeatCount, args)

		if counts[200] <= 0 {
			t.Fatalf("should be greater than 0")
		}
		for i := 0; i < 200; i++ {
			if counts[i] != 0 {
				t.Fatalf("should be 0")
			}
		}
	})

	t.Run("HitsDefaultMax", func(t *testing.T) {
		t.Parallel()

		regexp := "a*"
		args := &GeneratorArgs{
			RngSource: rand.NewSource(0),
		}
		counts := generateLenHistogram(regexp, DefaultMaxUnboundedRepeatCount, args)

		if len(counts) != DefaultMaxUnboundedRepeatCount+1 {
			t.Fatalf("should be equal")
		}
		if counts[DefaultMaxUnboundedRepeatCount] <= 0 {
			t.Fatalf("should be greater than 0")
		}
	})

	t.Run("HitsCustomMax", func(t *testing.T) {
		t.Parallel()

		regexp := "a*"
		args := &GeneratorArgs{
			RngSource:               rand.NewSource(0),
			MaxUnboundedRepeatCount: 200,
		}
		counts := generateLenHistogram(regexp, 200, args)

		if len(counts) != 200+1 {
			t.Fatalf("should be equal")
		}
		if counts[200] <= 0 {
			t.Fatalf("should be greater than 0")
		}
	})
}

func TestGenCharClassNotNl(t *testing.T) {
	t.Parallel()
	GeneratesStringMatchingItself(t, nil,
		"[a]",
		"[abc]",
		"[a-d]",
		"[ac]",
		"[0-9]",
		"[a-z0-9]",
	)

	// Try to narrow down the generation space. Still not a very strong assertion.
	generator, _ := NewGenerator("[^a-zA-Z0-9]", nil)
	for i := 0; i < SampleSize; i++ {
		if generator.Generate() == "\n" {
			t.Fatalf("should not include newline")
		}
	}
}

func TestGenNegativeCharClass(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil, "[^a-zA-Z0-9]")
}

func TestGenAlternate(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil,
		"a|b",
		"abc|def|ghi",
		"[ab]|[cd]",
		"foo|bar|baz", // rewrites to foo|ba[rz]
	)
}

func TestGenCapture(t *testing.T) {
	t.Parallel()

	GeneratesStringMatching(t, nil, "(abc)", "^abc$")
	GeneratesStringMatching(t, nil, "()", "^$")
}

func TestGenConcat(t *testing.T) {
	t.Parallel()

	GeneratesStringMatchingItself(t, nil, "[ab][cd]")
}

func TestGenRepeat(t *testing.T) {
	t.Parallel()

	t.Run("Unbounded", func(t *testing.T) {
		t.Parallel()

		GeneratesStringMatchingItself(t, nil, `a{1,}`)

		t.Run("HitsDefaultMax", func(t *testing.T) {
			t.Parallel()

			regexp := "a{0,}"
			args := &GeneratorArgs{
				RngSource: rand.NewSource(0),
			}
			counts := generateLenHistogram(regexp, DefaultMaxUnboundedRepeatCount, args)

			if len(counts) != DefaultMaxUnboundedRepeatCount+1 {
				t.Fatalf("should be equal")
			}
			if counts[DefaultMaxUnboundedRepeatCount] <= 0 {
				t.Fatalf("should be greater than 0")
			}
		})

		t.Run("HitsCustomMax", func(t *testing.T) {
			t.Parallel()

			regexp := "a{0,}"
			args := &GeneratorArgs{
				RngSource:               rand.NewSource(0),
				MaxUnboundedRepeatCount: 200,
			}
			counts := generateLenHistogram(regexp, 200, args)

			if len(counts) != 200+1 {
				t.Fatalf("should be equal")
			}
			if counts[200] <= 0 {
				t.Fatalf("should be greater than 0")
			}
		})
	})

	t.Run("HitsMin", func(t *testing.T) {
		t.Parallel()

		regexp := "a{0,3}"
		args := &GeneratorArgs{
			RngSource: rand.NewSource(0),
		}
		counts := generateLenHistogram(regexp, 3, args)

		if len(counts) != 3+1 {
			t.Fatalf("should be equal")
		}
		if counts[0] <= 0 {
			t.Fatalf("should be greater than 0")
		}
	})

	t.Run("HitsMax", func(t *testing.T) {
		t.Parallel()

		regexp := "a{0,3}"
		args := &GeneratorArgs{
			RngSource: rand.NewSource(0),
		}
		counts := generateLenHistogram(regexp, 3, args)

		if len(counts) != 3+1 {
			t.Fatalf("should be equal")
		}
		if counts[3] <= 0 {
			t.Fatalf("should be greater than 0")
		}
	})

	t.Run("IsWithinBounds", func(t *testing.T) {
		t.Parallel()

		regexp := "a{5,10}"
		args := &GeneratorArgs{
			RngSource: rand.NewSource(0),
		}
		counts := generateLenHistogram(regexp, 10, args)

		if len(counts) != 11 {
			t.Fatalf("should be equal")
		}

		for i := 0; i < 11; i++ {
			if i < 5 {
				if counts[i] != 0 {
					t.Fatalf("should equal 0 at %d", i)
				}
			} else if i < 11 {
				if counts[i] <= 0 {
					t.Fatalf("should >=0 at %d", i)
				}
			}
		}
	})
}

func TestGenCharClasses(t *testing.T) {
	t.Parallel()

	t.Run("Ascii", func(t *testing.T) {
		t.Parallel()

		GeneratesStringMatchingItself(t, nil,
			"[[:alnum:]]",
			"[[:alpha:]]",
			"[[:ascii:]]",
			"[[:blank:]]",
			"[[:cntrl:]]",
			"[[:digit:]]",
			"[[:graph:]]",
			"[[:lower:]]",
			"[[:print:]]",
			"[[:punct:]]",
			"[[:space:]]",
			"[[:upper:]]",
			"[[:word:]]",
			"[[:xdigit:]]",
			"[[:^alnum:]]",
			"[[:^alpha:]]",
			"[[:^ascii:]]",
			"[[:^blank:]]",
			"[[:^cntrl:]]",
			"[[:^digit:]]",
			"[[:^graph:]]",
			"[[:^lower:]]",
			"[[:^print:]]",
			"[[:^punct:]]",
			"[[:^space:]]",
			"[[:^upper:]]",
			"[[:^word:]]",
			"[[:^xdigit:]]",
		)
	})

	t.Run("Ascii", func(t *testing.T) {
		t.Parallel()

		args := &GeneratorArgs{
			Flags: syntax.Perl,
		}

		GeneratesStringMatchingItself(t, args,
			`\d`,
			`\s`,
			`\w`,
			`\D`,
			`\S`,
			`\W`,
		)
	})
}

func TestCaptureGroupHandler(t *testing.T) {
	t.Parallel()

	callCount := 0

	gen, err := NewGenerator(`(?:foo) (bar) (?P<name>baz)`, &GeneratorArgs{
		Flags: syntax.PerlX,
		CaptureGroupHandler: func(index int, name string, group *syntax.Regexp, generator Generator, args *GeneratorArgs) string {
			callCount++

			if index >= 2 {
				t.Fatalf("should be less 2")
			}

			if index == 0 {
				if name != "" {
					t.Fatalf("should be equal")
				}
				if group.String() != "bar" {
					t.Fatalf("should be equal")
				}
				if generator.Generate() != "bar" {
					t.Fatalf("should be equal")
				}
				return "one"
			}

			// Index 1

			if name != "name" {
				t.Fatalf("should be equal")
			}
			if group.String() != "baz" {
				t.Fatalf("should be equal")
			}
			if generator.Generate() != "baz" {
				t.Fatalf("should be equal")
			}
			return "two"
		},
	})
	if err != nil {
		t.Fatalf("error should be nil")
	}

	if gen.Generate() != "foo one two" {
		t.Fatalf("should be equal")
	}
	if callCount != 2 {
		t.Fatalf("should be equal")
	}
}

func GeneratesStringMatchingItself(t *testing.T, args *GeneratorArgs, patterns ...string) {
	for _, pattern := range patterns {
		s := ShouldGenerateStringMatching(pattern, pattern, args)
		if s != "" {
			t.Fatalf("String generated from /%s/ does not match itself", pattern)
		}
	}
}

func GeneratesStringMatching(t *testing.T, args *GeneratorArgs, pattern string, expectedPattern string) {
	s := ShouldGenerateStringMatching(pattern, expectedPattern, args)
	if s != "" {
		t.Fatalf("String generated from /%s/ does not match /%s/", pattern, expectedPattern)
	}
}

func ShouldGenerateStringMatching(actual interface{}, expected ...interface{}) string {
	return ShouldGenerateStringMatchingTimes(actual, expected[0], expected[1], SampleSize)
}

func ShouldGenerateStringMatchingTimes(actual interface{}, expected ...interface{}) string {
	pattern := actual.(string)
	expectedPattern := expected[0].(string)
	args := expected[1].(*GeneratorArgs)
	times := expected[2].(int)

	generator, err := NewGenerator(pattern, args)
	if err != nil {
		panic(err)
	}

	for i := 0; i < times; i++ {
		result := generator.Generate()
		matched, err := regexp.MatchString(expectedPattern, result)
		if err != nil {
			panic(err)
		}
		if !matched {
			return fmt.Sprintf("string “%s” generated from /%s/ did not match /%s/.",
				result, pattern, expectedPattern)
		}
	}

	return ""
}

func generateLenHistogram(regexp string, maxLen int, args *GeneratorArgs) (counts []int) {
	generator, err := NewGenerator(regexp, args)
	if err != nil {
		panic(err)
	}

	iterations := SampleSize
	if maxLen*4 > iterations {
		iterations = maxLen * 4
	}

	for i := 0; i < iterations; i++ {
		str := generator.Generate()

		// Grow the slice if necessary.
		if len(str) >= len(counts) {
			newCounts := make([]int, len(str)+1)
			copy(newCounts, counts)
			counts = newCounts
		}

		counts[len(str)]++
	}

	return
}
