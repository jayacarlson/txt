package txt

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/jayacarlson/dbg"
	"github.com/jayacarlson/rex"
)

// ================================================================================ //

/*
	Returns string with any latin1 characters (0x80..0xff) converted to
	their corrisponding rune -- unconvertables output as '-'
*/
var runes = []rune{
	//0x80,   0x81,   0x82,   0x83,   0x84,   0x85,   0x86,   0x87,
	//  €       -       ‚       ƒ       „       …       †       ‡
	0x20ac, 0x002d, 0x201a, 0x0192, 0x201e, 0x2026, 0x2020, 0x2021,

	//0x88,   0x89,   0x8a,   0x8b,   0x8c,   0x8d,   0x8e,   0x8f,
	//  ˆ       ‰       Š       ‹       Œ       -       Ž       -
	0x02c6, 0x2030, 0x0160, 0x2039, 0x0152, 0x002d, 0x017d, 0x002d,

	//0x90,   0x91,   0x92,   0x93,   0x94,   0x95,   0x96,   0x97,
	//  -       ‘       ’       “       ”       •       –       —
	0x002d, 0x2018, 0x2019, 0x201c, 0x201d, 0x2022, 0x2013, 0x2014,

	//0x98,   0x99,   0x9a,   0x9b,   0x9c,   0x9d,   0x9e,   0x9f,
	//  ˜       ™       š       ›       œ       -       ž       Ÿ
	0x02dc, 0x2122, 0x0161, 0x203a, 0x0153, 0x002d, 0x017e, 0x0178,
}

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

/*
	Clean up spaces inside a string (remove runs of spaces and replace with single)
	also trims leading / trailing whitespace
*/
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

/*
	Clean up a floating point string, remove any trailing ZEROs and . if needed
		e.g.	3.20000  -> 3.2     3.00000  -> 3
*/
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
	Convert escaped Unicode characters in a string to the actual character
		e.g. "BlahCo\u2122"  ->  "BlahCo™"
*/
var deUnicode = regexp.MustCompile(`((?s).*?)(\\u[[:xdigit:]]{4})((?s).*)`)

func FixUnicodeEscapedText(src string) string {
	return rex.RexReplace(src, deUnicode, func(x []string, rx *regexp.Regexp, rf rex.RexFunc) string {
		c, _ := strconv.Unquote(`"` + x[2] + `"`)
		return x[1] + fmt.Sprintf(c) + rex.RexReplace(x[3], rx, rf)
	})
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

/*
	Do <token> replacement in a string; <tokens> cannot contain other <tokens>
		-- unless handled as multiple calls from app
*/
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

/*
	Do {variable} replacement in a string; {variables} can contain other {variables}
	up to a max depth of 10 to prevent infinite loops
*/
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

// 	returns:	variable replaced string, any error, replacement occured
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

/*
	Take a newline seperated list and return a slice of strings
		e.g. the following list
			name1
			name2
			name3

		returns: []string{"name1", "name2", "name3"}

	Whitespace around the text is stripped
	Empty lines are ignored
	Any lines starting with the '#' inside the data area are stripped allowing commented text
*/
func ListToStringSlice(s string) []string {
	sl := strings.Split(s, "\n")
	ln := len(sl)
	for ix := 0; ix < ln; ix++ {
		sl[ix] = strings.TrimSpace(sl[ix])
		if sl[ix] == "" || sl[ix][0] == '#' {
			sl = append(sl[:ix], sl[ix+1:]...)
			ix--
			ln--
			continue
		}
	}
	return sl
}

/*
	Take a newline seperated list and return a single string with an arbitrary seperator
		e.g. the following list with a 'sep' of "; "
			name1
			name2
			name3

		returns: "name1; name2; name3"

	Whitespace around the text is stripped
	Empty lines are ignored
	Any lines starting with the '#' inside the data area are stripped allowing commented text
*/
func ListToSepString(str, sep string) string {
	return strings.Join(ListToStringSlice(str), sep)
}

/*
	Read in configuration information looking for a 'label' and an area of text
	 surrounded by {braces} that would be the 'config' data; ending bracket must
	 be 1st and only character on closing line

	The data contained inside the braces can have an optional seperator character
	 (,;:|/) which is replaced with a newline character

	On finding config information, the supplied function is called with the label
	 and data read

	The returned config string could then be passed to ListToSepString or ListToSepStringSlice
	 if desired

		e.g. given some sample text of:

		myLabel1 {
			data1.1
			data1.2
		}

		myLabel2 , {
			data2.1, data2.2
		}

	Callback func would be called with "myLabel1", `
	data1.1
	data1.2
	`

	Then called with "myLabel2", `
	data2.1
	 data2.2			# NOTE the space after the , is not removed from the source text "data2.1, data2.2"
	`

	Whitespace around the text is stripped
	Empty lines are ignored
	Any lines starting with the '#' inside the data area are stripped allowing commented text
*/
var infoRex = regexp.MustCompile("((?ms).*?^(\\w+)\\s*([,;:|/]?)\\s*{)((?s).*?)\n}\n((?s).*)")

func HandleConfigData(str string, f func(label, data string)) {
	for x := infoRex.FindStringSubmatch(str); x != nil; x = infoRex.FindStringSubmatch(x[5]) {
		x[4] = ListToSepString(x[4], "\n")
		if x[3] != "" {
			x[4] = strings.Replace(x[4], x[3], "\n", -1)
		}
		f(x[2], x[4])
	}
}

/*
	Reads the config file and passes returned data to handler
*/
func LoadConfigData(flPath string, f func(label, data string)) {
	data, err := ioutil.ReadFile(flPath)
	dbg.ChkErrX(err, "Failed to read config file: %s (%v)", flPath, err)
	HandleConfigData(string(data), f)
}

/*
	Grabs a block of labeled config data between < & > characters; note the
	ending '>' must be the 1st and only character on the closing line:

	block1<
	blah blah
	blah blah
	>

	block2 <more stuff...
	>

	All text between the < & > characters is captured, including leading / trailing whitespace;
	note the last \n is consumed, e.g. block2 is returned as 'more stuff...'
*/
var blocRex = regexp.MustCompile("((?ms).*?^(\\w+)\\s*<)((?s).*?)\n>\n((?s).*)")

func HandleConfigBlocks(str string, f func(label, data string)) {
	for x := blocRex.FindStringSubmatch(str); x != nil; x = blocRex.FindStringSubmatch(x[4]) {
		f(x[2], x[3])
	}
}

/*
	Reads the config file and passes returned data to handler
*/
func LoadConfigBlocks(flPath string, f func(label, data string)) {
	data, err := ioutil.ReadFile(flPath)
	dbg.ChkErrX(err, "Failed to read config file: %s (%v)", flPath, err)
	HandleConfigBlocks(string(data), f)
}
