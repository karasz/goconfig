// Package goconfig parses .gitconfig style files
package goconfig

import (
	"unicode"
)

type parser struct {
	runes  []rune
	linenr uint
	eof    bool
}

// Parse takes given bytes as configuration file (according to gitconfig syntax)
func Parse(bytes []byte) (map[string]string, uint, error) {
	parser := &parser{[]rune(string(bytes)), 1, false}
	cfg, err := parser.parse()
	return cfg, parser.linenr, err
}

func (cf *parser) parse() (map[string]string, error) {
	comment := false
	cfg := map[string]string{}
	name := ""
	var err error
	for {
		c := cf.nextRune()
		if c == '\n' {
			if cf.eof {
				return cfg, nil
			}
			comment = false
			continue
		}
		if comment || isspace(c) {
			continue
		}
		if c == '#' || c == ';' {
			comment = true
			continue
		}
		if c == '[' {
			name, err = cf.getSectionKey()
			if err != nil {
				return cfg, err
			}
			name += "."
			continue
		}
		if !isalpha(c) {
			return cfg, ErrInvalidKeyChar
		}
		key := name + string(c)
		value, err := cf.getValue(&key)
		if err != nil {
			return cfg, err
		}
		cfg[key] = value
	}
}

func (cf *parser) nextRune() rune {
	if len(cf.runes) == 0 {
		cf.eof = true
		return '\n'
	}
	c := cf.runes[0]
	if c == '\r' {
		/* DOS like systems */
		if len(cf.runes) > 1 && cf.runes[1] == '\n' {
			cf.runes = cf.runes[1:]
			c = '\n'
		}
	}
	if c == '\n' {
		cf.linenr++
	}
	if len(cf.runes) == 0 {
		cf.eof = true
		cf.linenr++
		c = '\n'
	}
	cf.runes = cf.runes[1:]
	return c
}

func (cf *parser) getSectionKey() (string, error) {
	name := ""
	for {
		c := cf.nextRune()
		if cf.eof {
			return "", ErrUnexpectedEOF
		}
		if c == ']' {
			return name, nil
		}
		if isspace(c) {
			return cf.getExtendedSectionKey(name, c)
		}
		if !iskeychar(c) && c != '.' {
			return "", ErrInvalidSectionChar
		}
		name += string(lower(c))
	}
}

// config: [BaseSection "ExtendedSection"]
func (cf *parser) getExtendedSectionKey(name string, c rune) (string, error) {
	for {
		if c == '\n' {
			cf.linenr--
			return "", ErrSectionNewLine
		}
		c = cf.nextRune()
		if !isspace(c) {
			break
		}
	}
	if c != '"' {
		return "", ErrMissingStartQuote
	}
	name += "."
	for {
		c = cf.nextRune()
		if c == '\n' {
			cf.linenr--
			return "", ErrSectionNewLine
		}
		if c == '"' {
			break
		}
		if c == '\\' {
			c = cf.nextRune()
			if c == '\n' {
				cf.linenr--
				return "", ErrSectionNewLine
			}
		}
		name += string(c)
	}
	if cf.nextRune() != ']' {
		return "", ErrMissingClosingBracket
	}
	return name, nil
}

func (cf *parser) getValue(name *string) (string, error) {
	var c rune
	var err error
	var value string

	/* Get the full name */
	for {
		c = cf.nextRune()
		if cf.eof {
			break
		}
		if !iskeychar(c) {
			break
		}
		*name += string(lower(c))
	}

	for c == ' ' || c == '\t' {
		c = cf.nextRune()
	}

	if c != '\n' {
		if c != '=' {
			return "", ErrInvalidKeyChar
		}
		value, err = cf.parseValue()
		if err != nil {
			return "", err
		}
	}
	return value, err
}

func (cf *parser) parseValue() (string, error) {
	var quote, comment bool
	var space int

	var value string

	// strbuf_reset(&cf->value);
	for {
		c := cf.nextRune()
		if c == '\n' {
			if quote {
				cf.linenr--
				return "", ErrUnfinishedQuote
			}
			return value, nil
		}
		if comment {
			continue
		}
		if isspace(c) && !quote {
			if len(value) > 0 {
				space++
			}
			continue
		}
		if !quote {
			if c == ';' || c == '#' {
				comment = true
				continue
			}
		}
		for space != 0 {
			value += " "
			space--
		}
		if c == '\\' {
			c = cf.nextRune()
			switch c {
			case '\n':
				continue
			case 't':
				c = '\t'
			case 'b':
				c = '\b'
			case 'n':
				c = '\n'
			default:
				return "", ErrInvalidEscapeSequence
			}
			value += string(c)
			continue
		}
		if c == '"' {
			quote = !quote
			continue
		}
		value += string(c)
	}
}

func lower(c rune) rune {
	return unicode.ToLower(c)
}

func isspace(c rune) bool {
	return unicode.IsSpace(c)
}

func iskeychar(c rune) bool {
	return isalnum(c) || c == '-'
}

func isalnum(c rune) bool {
	return isalpha(c) || isnum(c)
}

func isalpha(c rune) bool {
	return unicode.IsLetter(c)
}

func isnum(c rune) bool {
	return unicode.IsNumber(c)
}
