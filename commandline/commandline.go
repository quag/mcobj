package commandline

import (
	"strings"
	"unicode/utf8"
)

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
	input string
	state stateFn
	pos   int
	start int
	width int
	args  []string
	frags []string
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) peek() (r rune) {
	r = l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) emit() {
	l.frags = append(l.frags, l.input[l.start:l.pos])
	l.start = l.pos
}

func (l *lexer) emitArg() {
	l.args = append(l.args, strings.Join(l.frags, ""))
	l.frags = l.frags[:0]
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func lex(input string) *lexer {
	return &lexer{
		input: input,
		state: lexBetweenArg,
		args:  make([]string, 0),
		frags: make([]string, 0),
	}
}

func lexBetweenArg(l *lexer) stateFn {
	l.acceptRun(" ")
	l.ignore()
	if l.next() != eof {
		l.backup()
		return lexArg
	}
	return nil
}

func lexArg(l *lexer) stateFn {
	next := stateFn(nil)
	endArg := false

Loop:
	for {
		switch l.next() {
		case '"':
			next = lexDblQuoteArg
		case '\'':
			next = lexSglQuoteArg
		case ' ':
			next = lexBetweenArg
			endArg = true
		case '\\':
			start := l.pos - l.width
			switch l.next() {
			case ' ':
				tmp := l.pos
				l.pos = start
				if l.pos > l.start {
					l.emit()
				}
				l.start = tmp - l.width
				l.pos = tmp
				continue
			case '\'':
				continue
			case '"':
				continue
			case eof:
				// TODO: Error, incomplete escape
				continue
			default:
				// TODO: Error, unknown escape
				continue
			}
		case eof:
			next = nil
			endArg = true
		default:
			continue
		}
		break Loop
	}
	l.backup()
	if l.pos > l.start {
		l.emit()
	}
	if endArg {
		l.emitArg()
	}
	return next
}

func lexDblQuoteArg(l *lexer) stateFn {
	return lexQuoteArg(l, '"')
}

func lexSglQuoteArg(l *lexer) stateFn {
	return lexQuoteArg(l, '\'')
}

func lexQuoteArg(l *lexer, quote rune) stateFn {
	l.next()
	l.ignore()
Loop:
	for {
		switch l.next() {
		case '\\':
			start := l.pos - l.width
			switch l.next() {
			case ' ':
				tmp := l.pos
				l.pos = start
				if l.pos > l.start {
					l.emit()
				}
				l.start = tmp - l.width
				l.pos = tmp
				continue
			case '\'':
			case '"':
			case eof:
				// TODO: Error, incomplete escape
			default:
				// TODO: Error, unknown escape
			}
		case eof:
			// error unterminated quote
			return lexArg
		case quote:
			break Loop
		}
	}
	l.backup()
	l.emit()
	l.next()
	l.ignore()
	return lexArg
}

func SplitCommandLine(commandLine string) []string {
	l := lex(commandLine)
	for l.state != nil {
		l.state = l.state(l)
	}
	return l.args
}
