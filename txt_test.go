package txt

import (
	"io/ioutil"
	"runtime"
	"strings"
	"testing"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/pth"
)

var (
	tokens    = make(TokenizerMap)
	variables = make(VariableMap)

	templateT    = `{ "customer": <num>, "name": "<name>", "phone": "<phone>" }`
	templateTBad = `{ "customer": <num>, "name": "<name>", "phone": "<phone>", "bad": "<bad>" }`
	templateV    = `Name: {name}, Age: {age}, Addr: {addr}, Phone: {phone}`
	templateVBad = `Name: {name}, Age: {age}, Addr: {addr}, Phone: {phone}, Bad1: {bad1}, Bad2: {bad2}`
)

func init() {
	tokens["num"] = "Num-Token"
	tokens["name"] = "Name-Token"
	tokens["phone"] = "Phone-Token"

	variables["getName"] = "Name-Result"
	variables["name"] = "{getName}"
	variables["getAge"] = "Age-Result"
	variables["getGetAge"] = "{getAge}"
	variables["age"] = "{getGetAge}"
	variables["addr"] = "Addr-Result"
	variables["phone"] = "Phone-Result"
}

func iAm() string {
	pc := make([]uintptr, 4)
	runtime.Callers(2, pc)
	nm := runtime.FuncForPC(pc[0]).Name()
	return nm[strings.LastIndex(nm, ".")+1:]
}

func TestRuneizer(t *testing.T) {
	expected := "pqrstuvwxyz{|}~-€-‚ƒ„…†‡ˆ‰Š‹Œ-Ž--‘’“”•–—˜™š›œ-žŸ ¡¢£¤¥¦§¨©ª«¬-®¯"
	text := []byte{
		0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
		0x78, 0x79, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f,

		0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87,
		// €     -     ‚     ƒ     „     …     †     ‡
		0x88, 0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f,
		// ˆ     ‰     Š     ‹     Œ     -     Ž     -
		0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97,
		// -     ‘     ’     “     ”     •     –     —
		0x98, 0x99, 0x9a, 0x9b, 0x9c, 0x9d, 0x9e, 0x9f,
		// ˜     ™     š     ›     œ     -     ž     Ÿ

		0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7,
		0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf,
	}
	result := Latin1Runeizer(string(text))
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

func TestCleanSpaces(t *testing.T) {
	expected := "This Is A Test Of Cleaning Spaces"
	text := `

	This Is  A   Test    Of   Cleaning  Spaces


    `
	result := CleanSpaces(text)
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

func TestTrimDot0sA(t *testing.T) {
	expected := "3.2"
	result := TrimDot0s("3.20000")
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

func TestTrimDot0sB(t *testing.T) {
	expected := "3"
	result := TrimDot0s("3")
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

func TestTrimDot0sC(t *testing.T) {
	expected := "0"
	result := TrimDot0s("-0.0000")
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

func TestTokenizerA(t *testing.T) {
	expected := `{ "customer": Num-Token, "name": "Name-Token", "phone": "Phone-Token" }`
	result, _ := tokens.DeTokenize(templateT)
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

func TestTokenizerB(t *testing.T) {
	expected := `{ "customer": Num-Token, "name": "Name-Token", "phone": "Phone-Token", "bad": "!BAD-TOKEN:'bad'!" }`
	expErr := `DeTokenize: <bad> unknown`
	result, resErr := tokens.DeTokenize(templateTBad)
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
	if resErr.Error() != expErr {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Caution("Expected Error  >%s<", expErr)
		dbg.Error("Resulting Error >%v<", resErr)
	}
}

func TestVariableReplacementA(t *testing.T) {
	// expected result is all var names replaced, even through other variables
	expected := `Name: Name-Result, Age: Age-Result, Addr: Addr-Result, Phone: Phone-Result`
	result, _ := variables.ReplaceVars(templateV)
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

func TestVariableReplacementB(t *testing.T) {
	// expected result is a bad variable '{bad1}' is not matched, result will contain
	// incomplete variable replacements as operation was halted at failure, error returned
	expected := `Name: {getName}, Age: {getGetAge}, Addr: Addr-Result, Phone: Phone-Result, Bad1: !BAD-VAR:'bad1'!, Bad2: !BAD-VAR:'bad2'!`
	expErr := `ReplaceVars: {bad1} unknown`
	result, resErr := variables.ReplaceVars(templateVBad)
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
	if resErr.Error() != expErr {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Caution("Expected Error  >%s<", expErr)
		dbg.Error("Resulting Error >%v<", resErr)
	}
}

func TestUnicodeCorrection(t *testing.T) {
	testString := `Test \u0031 \u0026 \u0032 then some letters \u00c1 \u00e1 \u00e2 \u00e3 \u00e4 \u00e5 \u00c9 \u00e8 \u00e9 \u00ea \u00eb \u00ed \u00ee \u00ef \u00d3 \u00d8 \u00f8 \u00f3 \u00f4 \u00f6 \u00f9 \u00fa \u00fc \u00e7 \u00f1`
	expected := `Test 1 & 2 then some letters Á á â ã ä å É è é ê ë í î ï Ó Ø ø ó ô ö ù ú ü ç ñ`
	result := FixUnicodeEscapedText(testString)
	if result != expected {
		t.Fail()
		dbg.Danger("Test Failure in: %s", iAm())
		dbg.Info("Expected >%s<", expected)
		dbg.Error("Result   >%s<", result)
	}
}

const (
	blk1 = `
block1 blah blah
blah blah blah`
	blk2 = `
block2 blah blah
blah blah blah`
	blk3 = `block3 blah blah
blah blah blah`
	dta1 = `apple
banana
cherry
date
{ }
} {
fig
grape`
	dta2 = `apple
banana
cherry
date
fig
grape`
)

func TestBlockConfig(t *testing.T) {
	conf, err := ioutil.ReadFile(pth.AsRealPath("$/testdata/testBlocks.txt"))
	dbg.ChkErrX(err, "Failed to read config file", t.Fail)

	HandleConfigBlocks(string(conf), func(l, d string) {
		switch l {
		case "block1":
			if d != blk1 {
				dbg.Error("block1:\n<%s>", d)
				t.Fail()
			}
		case "block2":
			if d != blk2 {
				dbg.Error("block2:\n<%s>", d)
				t.Fail()
			}
		case "block3":
			if d != blk3 {
				dbg.Error("block3:\n<%s>", d)
				t.Fail()
			}
		default:
			if l != "unknownBlock" {
				dbg.Error("Unknown block: %s", l)
				t.Fail()
			}

		}
	})

	HandleConfigData(string(conf), func(l, d string) {
		switch l {
		case "data1":
			if d != dta1 {
				dbg.Error("data1:\n%s", d)
				t.Fail()
			}
		case "data2":
			fd := ListToSepString(d, "\n")
			if fd != dta2 {
				dbg.Error("data2:\n%s", d)
				t.Fail()
			}
		default:
			dbg.Error("Unknown block: %s", l)
			t.Fail()

		}
	})
}
