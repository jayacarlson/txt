package txt

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ================================================================================ //

var runes = []rune{
	//0x80,   0x81,   0x82,   0x83,   0x84,   0x85,   0x86,   0x87,
	0x20ac, 0x002d, 0x201a, 0x0192, 0x201e, 0x2026, 0x2020, 0x2021,
	//  €       -       ‚       ƒ       „       …       †       ‡

	//0x88,   0x89,   0x8a,   0x8b,   0x8c,   0x8d,   0x8e,   0x8f,
	0x02c6, 0x2030, 0x0160, 0x2039, 0x0152, 0x002d, 0x017d, 0x002d,
	//  ˆ       ‰       Š       ‹       Œ       -       Ž       -

	//0x90,   0x91,   0x92,   0x93,   0x94,   0x95,   0x96,   0x97,
	0x002d, 0x2018, 0x2019, 0x201c, 0x201d, 0x2022, 0x2013, 0x2014,
	//  -       ‘       ’       “       ”       •       –       —

	//0x98,   0x99,   0x9a,   0x9b,   0x9c,   0x9d,   0x9e,   0x9f,
	0x02dc, 0x2122, 0x0161, 0x203a, 0x0153, 0x002d, 0x017e, 0x0178,
	//  ˜       ™       š       ›       œ       -       ž       Ÿ
}

// returns string with any latin1 characters (0x80..0xff) converted to
//	their corrisponding rune -- unconvertables output as '-'
func Latin1Runeizer(str string) string {
	var sba = []byte(str)
	var buf = bytes.NewBuffer(make([]byte, 0))
	var r rune

	for _, b := range sba {
		switch {
		case b >= 0x80 && b < 0xA0:
			r = runes[b-0x80]
		case b == 0x7F, b == 0xAD:
			r = rune('-')
		default:
			r = rune(b)
		}
		buf.WriteRune(r)
	}
	return string(buf.Bytes())
}

// clean up spaces inside a string (remove runs of spaces and replace with single ' ')
//	also trims leading / trailing whitespace
func CleanSpaces(str string) string {
	spc := false
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if spc && r == ' ' {
			return -1
		}
		spc = r == ' '
		return r
	}, str))
}

// clean up a floating point string, remove any trailing ZEROs and . if needed
//	e.g.	3.20000  -> 3.2     3.00000  -> 3
func TrimDot0s(o string) string {
	if -1 != strings.Index(o, ".") {
		o = strings.TrimRight(strings.TrimRight(o, "0"), ".")
	}
	if o == "-0" {
		o = "0"
	}
	return o
}

// convert and trim floating point value
func FltTrimDot0s(f float64) string {
	return TrimDot0s(fmt.Sprintf("%f", f))
}

/*
Some routines to do text replacement inside template strings using string maps

<Token> replacement is done by wrapping a keyword with the '<' & '>' characters
{Variable} replacement is done by wrapping a keyword with the '{' & '}' characters

The only difference between them (other than the wrapping) is {variable} text
can contain additional {variables} -- just as something I was playing around with
thinking it might be handy.

	var tok = make(TokenizerMap)
	var templateT = `Name: <name>\nAge: <age>\nAddr: <addr>\nPhone: <phone>`
	var templateJ = `{\t"customer": <num>,\n\t"name": "<name>",\n\t"phone": "<phone>"\n}`

	// return info as text for printing and a json version for saving to file
	func output(num int, name, addr, phone string, age int ) (string, string) {
		tok["name"] = name
		tok["addr"] = addr
		tok["phone"] = phone
		tok["num"] = strconv.Itoa(num)
		tok["age"] = strconv.Itoa(age)
		return tok.DeTokenize(templateT), tok.DeTokenize(templateJ)
	}
*/

type (
	TokenizerMap map[string]string
	VariableMap  map[string]string
)

var (
	tokRex       = regexp.MustCompile("((?s).*?)<([[:alnum:]]+?)>((?s).*)")
	vrmRex       = regexp.MustCompile("((?s).*?){([[:alnum:]]+?)}((?s).*)")
	Err_MaxDepth = errors.New("ReplaceVars: max depth")
)

// do <token> replacement in a string; <tokens> cannot contain other <tokens>
//	-- unless handled as multiple calls from app
func (tok TokenizerMap) DeTokenize(src string) (string, error) {
	var err error = nil
	if x := tokRex.FindStringSubmatch(src); x != nil {
		vl, ok := tok[x[2]]
		if !ok {
			vl = "!BAD-TOKEN:'" + x[2] + "'!"
			err = errors.New("DeTokenize: <" + x[2] + "> unknown")
		}
		ss, lerr := tok.DeTokenize(x[3])
		if err == nil && lerr != nil {
			err = lerr
		}
		return x[1] + vl + ss, err
	}
	return src, nil
}

// do {variable} replacement in a string; {variables} can contain other {variables}
//	up to a max depth of 10 to prevent infinite loops
func (vrm VariableMap) ReplaceVars(src string) (string, error) {
	var err error = nil
	rep, depth := true, 0
	for rep {
		if depth >= 10 {
			return src, Err_MaxDepth
		}
		depth += 1
		src, err, rep = vrm.varRep(src)
		if err != nil {
			return src, err
		}
	}
	return src, nil
}

// 	returns:	variable replaced string, replacement occured, error occured
func (vrm VariableMap) varRep(src string) (string, error, bool) {
	if x := vrmRex.FindStringSubmatch(src); x != nil {
		var err error = nil
		vl, ok := vrm[x[2]]
		if !ok {
			vl = "!BAD-VAR:'" + x[2] + "'!"
			err = errors.New("ReplaceVars: {" + x[2] + "} unknown")
		}
		ss, lerr, _ := vrm.varRep(x[3])
		if err == nil && lerr != nil {
			err = lerr
		}
		return x[1] + vl + ss, err, true
	}
	return src, nil, false
}
