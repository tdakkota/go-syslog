package rfc6587

import (
	syslog "github.com/influxdata/go-syslog"
	"github.com/influxdata/go-syslog/rfc5424"
	parser "github.com/leodido/ragel-machinery/parser"
	"io"
)

const rfc6587Start int = 1
const rfc6587Error int = 0

const rfc6587EnMain int = 1

type machine struct {
	trailertyp TrailerType // default is 0 thus TrailerType(LF)
	trailer    byte
	candidate  []byte
	bestEffort bool
	internal   syslog.Machine
	emit       syslog.ParserListener
}

// Exec implements the ragel.Parser interface.
func (m *machine) Exec(s *parser.State) (int, int) {
	// Retrieve previously stored parsing variables
	cs, p, pe, eof, data := s.Get()
	// Inline FSM code here

	{
		var _widec int16
		if p == pe {
			goto _testEof
		}
		switch cs {
		case 1:
			goto stCase1
		case 0:
			goto stCase0
		case 2:
			goto stCase2
		case 3:
			goto stCase3
		}
		goto stOut
	stCase1:
		if data[p] == 60 {
			goto tr0
		}
		goto st0
	stCase0:
	st0:
		cs = 0
		goto _out
	tr0:

		if len(m.candidate) > 0 {
			m.process()
		}
		m.candidate = make([]byte, 0)

		goto st2
	st2:
		if p++; p == pe {
			goto _testEof2
		}
	stCase2:
		_widec = int16(data[p])
		switch {
		case data[p] > 0:
			if 10 <= data[p] && data[p] <= 10 {
				_widec = 256 + (int16(data[p]) - 0)
				if m.trailertyp == LF {
					_widec += 256
				}
			}
		default:
			_widec = 768 + (int16(data[p]) - 0)
			if m.trailertyp == NUL {
				_widec += 256
			}
		}
		switch _widec {
		case 266:
			goto st2
		case 522:
			goto tr3
		case 768:
			goto st2
		case 1024:
			goto tr3
		}
		switch {
		case _widec > 9:
			if 11 <= _widec {
				goto st2
			}
		case _widec >= 1:
			goto st2
		}
		goto st0
	tr3:

		m.candidate = append(m.candidate, data...)

		goto st3
	st3:
		if p++; p == pe {
			goto _testEof3
		}
	stCase3:
		_widec = int16(data[p])
		switch {
		case data[p] > 0:
			if 10 <= data[p] && data[p] <= 10 {
				_widec = 256 + (int16(data[p]) - 0)
				if m.trailertyp == LF {
					_widec += 256
				}
			}
		default:
			_widec = 768 + (int16(data[p]) - 0)
			if m.trailertyp == NUL {
				_widec += 256
			}
		}
		switch _widec {
		case 60:
			goto tr0
		case 266:
			goto st2
		case 522:
			goto tr3
		case 768:
			goto st2
		case 1024:
			goto tr3
		}
		switch {
		case _widec > 9:
			if 11 <= _widec {
				goto st2
			}
		case _widec >= 1:
			goto st2
		}
		goto st0
	stOut:
	_testEof2:
		cs = 2
		goto _testEof
	_testEof3:
		cs = 3
		goto _testEof

	_testEof:
		{
		}
	_out:
		{
		}
	}

	// Update parsing variables
	s.Set(cs, p, pe, eof)
	return p, pe
}

func (m *machine) OnErr() {
	// todo(leodido) > handle unexpected errors (only unexepected EOFs?)
}

func (m *machine) OnEOF() {
}

func (m *machine) OnCompletion() {
	if len(m.candidate) > 0 {
		m.process()
	}
}

// NewParser returns a syslog.Parser suitable to parse syslog messages sent with non-transparent framing - ie. RFC 6587.
func NewParser(options ...syslog.ParserOption) syslog.Parser {
	m := &machine{
		emit: func(*syslog.Result) { /* noop */ },
	}

	for _, opt := range options {
		m = opt(m).(*machine)
	}

	// No error can happens since during its setting we check the trailer type passed in
	trailer, _ := m.trailertyp.Value()
	m.trailer = byte(trailer)

	if m.internal == nil {
		m.internal = rfc5424.NewMachine()
	}

	return m
}

// HasBestEffort tells whether the receiving parser has best effort mode on or off.
func (m *machine) HasBestEffort() bool {
	return m.bestEffort
}

// WithTrailer ... todo(leodido)
func WithTrailer(t TrailerType) syslog.ParserOption {
	return func(m syslog.Parser) syslog.Parser {
		if val, err := t.Value(); err == nil {
			m.(*machine).trailer = byte(val)
			m.(*machine).trailertyp = t
		}
		return m
	}
}

// WithBestEffort sets the best effort mode on.
//
// When active the parser tries to recover as much of the syslog messages as possible.
func WithBestEffort(f syslog.ParserListener) syslog.ParserOption {
	return func(m syslog.Parser) syslog.Parser {
		var p = m.(*machine)
		p.bestEffort = true
		// Push down the best effort, too
		p.internal = rfc5424.NewParser(rfc5424.WithBestEffort())
		return p
	}
}

// WithListener specifies the function to send the results of the parsing.
func WithListener(f syslog.ParserListener) syslog.ParserOption {
	return func(m syslog.Parser) syslog.Parser {
		machine := m.(*machine)
		machine.emit = f
		return machine
	}
}

// Parse parses the io.Reader incoming bytes.
//
// It stops parsing when an error regarding RFC 6587 is found.
func (m *machine) Parse(reader io.Reader) {
	r := parser.ArbitraryReader(reader, m.trailer)
	parser.New(r, m, parser.WithStart(1)).Parse()
}

func (m *machine) process() {
	lastOne := len(m.candidate) - 1
	if m.candidate[lastOne] == m.trailer {
		m.candidate = m.candidate[:lastOne]
	}
	res, err := m.internal.Parse(m.candidate)
	m.emit(&syslog.Result{
		Message: res,
		Error:   err,
	})
}

// todo(leodido) > error management.
