package defndb

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"unsafe"
)

// The index cache's wire format.
//
// gob was the obvious first choice and measured as a net loss: 34 MB and 357ms
// to encode, 136ms to decode, against a 193ms parse of the same corpus. A cache
// that costs more than the work it replaces is not a cache.
//
// The cost is structural, not gob's fault. An index record is six mostly-
// repeated strings: DefName repeats once per field in a var, TypeName is one of
// a few dozen predicate names, FieldName is one of about ten, SourceFile is
// constant within a fragment, and even FieldValue repeats — a claim's Subject
// and Object are var names that appear over and over. Any encoding that writes
// those inline pays for the same bytes 132733 times.
//
// So strings are written once into a table and referenced by varint index
// everywhere else. That is the whole trick, and it takes the payload from 34 MB
// to a few MB with a decode dominated by reading the table rather than
// materialising records.
//
// The format is winze-internal and versioned by cacheVersion; there is no
// compatibility obligation. A cache that fails to decode for any reason is
// discarded and rebuilt, so a format change needs only a version bump.

var errBadCache = errors.New("defndb: malformed cache")

// stringTable assigns stable indices to strings during encoding.
type stringTable struct {
	idx  map[string]uint64
	list []string
}

func newStringTable() *stringTable {
	return &stringTable{idx: map[string]uint64{}}
}

func (t *stringTable) id(s string) uint64 {
	if i, ok := t.idx[s]; ok {
		return i
	}
	i := uint64(len(t.list))
	t.idx[s] = i
	t.list = append(t.list, s)
	return i
}

type encoder struct {
	w   *bufio.Writer
	buf []byte
}

func (e *encoder) uint(v uint64) {
	e.buf = binary.AppendUvarint(e.buf[:0], v)
	e.w.Write(e.buf)
}

func (e *encoder) int(v int) { e.uint(uint64(v)) }

// encodeCache writes the whole cache. Fragments are written in sorted path
// order so the file is byte-identical for an unchanged corpus, which keeps the
// cache itself diffable when someone goes looking.
func encodeCache(w io.Writer, dir string, paths []string, frags map[string]fragment) error {
	t := newStringTable()

	// Pass 1: intern every string, recording nothing. The table must be
	// complete before any record referencing it is written.
	t.id(dir)
	for _, p := range paths {
		f, ok := frags[p]
		if !ok {
			continue
		}
		t.id(p)
		for _, rt := range f.RoleTypes {
			t.id(rt.Name)
			t.id(rt.SourceFile)
		}
		for _, d := range f.Defs {
			t.id(d.Name)
			t.id(d.Kind)
			t.id(d.SourceFile)
		}
		for _, c := range f.Candidates {
			t.id(c.VarName)
			t.id(c.RoleType)
			t.id(c.SourceFile)
		}
		for _, l := range f.Literals {
			t.id(l.DefName)
			t.id(l.TypeName)
			t.id(l.FieldName)
			t.id(l.FieldValue)
			t.id(l.SourceFile)
		}
		for _, p := range f.Pragmas {
			t.id(p.DefName)
			t.id(p.SourceFile)
			t.id(p.Key)
			t.id(p.Value)
		}
	}

	bw := bufio.NewWriterSize(w, 1<<20)
	e := &encoder{w: bw, buf: make([]byte, 0, binary.MaxVarintLen64)}

	e.int(cacheVersion)
	e.int(len(t.list))
	for _, s := range t.list {
		e.int(len(s))
		bw.WriteString(s)
	}
	e.uint(t.id(dir))

	var present []string
	for _, p := range paths {
		if _, ok := frags[p]; ok {
			present = append(present, p)
		}
	}
	e.int(len(present))
	for _, p := range present {
		f := frags[p]
		e.uint(t.id(p))
		e.uint(uint64(f.Key.Size))
		e.uint(uint64(f.Key.MTime))

		e.int(len(f.RoleTypes))
		for _, rt := range f.RoleTypes {
			e.uint(t.id(rt.Name))
			e.uint(t.id(rt.SourceFile))
		}
		e.int(len(f.Defs))
		for _, d := range f.Defs {
			e.uint(t.id(d.Name))
			e.uint(t.id(d.Kind))
			e.uint(t.id(d.SourceFile))
			e.int(d.Line)
		}
		e.int(len(f.Candidates))
		for _, c := range f.Candidates {
			e.uint(t.id(c.VarName))
			e.uint(t.id(c.RoleType))
			e.uint(t.id(c.SourceFile))
		}
		e.int(len(f.Literals))
		for _, l := range f.Literals {
			e.uint(t.id(l.DefName))
			e.uint(t.id(l.TypeName))
			e.uint(t.id(l.FieldName))
			e.uint(t.id(l.FieldValue))
			e.uint(t.id(l.SourceFile))
			e.int(l.Line)
		}
		e.int(len(f.Pragmas))
		for _, pr := range f.Pragmas {
			e.uint(t.id(pr.DefName))
			e.uint(t.id(pr.SourceFile))
			e.int(pr.Line)
			e.uint(t.id(pr.Key))
			e.uint(t.id(pr.Value))
		}
	}
	return bw.Flush()
}

type decoder struct {
	data []byte
	pos  int
	tab  []string
	err  error
}

func (d *decoder) uint() uint64 {
	if d.err != nil {
		return 0
	}
	v, n := binary.Uvarint(d.data[d.pos:])
	if n <= 0 {
		d.err = errBadCache
		return 0
	}
	d.pos += n
	return v
}

func (d *decoder) int() int { return int(d.uint()) }

// str resolves a table index. An out-of-range index means a truncated or
// corrupt file, which is a discard-and-rebuild, never a panic.
func (d *decoder) str() string {
	i := d.uint()
	if d.err != nil {
		return ""
	}
	if i >= uint64(len(d.tab)) {
		d.err = errBadCache
		return ""
	}
	return d.tab[i]
}

// decodeCache reads a cache written by encodeCache. The whole file is read into
// memory first: it is a few MB and the alternative is a syscall per varint.
//
// Table strings are sliced out of that single buffer rather than copied, so the
// decode allocates one large backing array instead of one small string per
// entry. The buffer is therefore retained for the life of the index, which is
// the intended trade — the index outlives nothing, since the process exits
// right after answering.
func decodeCache(r io.Reader) (string, map[string]fragment, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", nil, err
	}
	d := &decoder{data: data}

	if v := d.int(); v != cacheVersion {
		return "", nil, errBadCache
	}
	n := d.int()
	if d.err != nil || n < 0 || n > len(data) {
		return "", nil, errBadCache
	}
	d.tab = make([]string, n)
	for i := 0; i < n; i++ {
		ln := d.int()
		if d.err != nil || ln < 0 || d.pos+ln > len(data) {
			return "", nil, errBadCache
		}
		d.tab[i] = unsafeString(data[d.pos : d.pos+ln])
		d.pos += ln
	}
	dir := d.str()

	count := d.int()
	if d.err != nil || count < 0 || count > len(data) {
		return "", nil, errBadCache
	}
	frags := make(map[string]fragment, count)
	for i := 0; i < count; i++ {
		path := d.str()
		var f fragment
		f.Key.Size = int64(d.uint())
		f.Key.MTime = int64(d.uint())

		if n := d.int(); n > 0 {
			if d.err != nil || n > len(data) {
				return "", nil, errBadCache
			}
			f.RoleTypes = make([]RoleType, n)
			for j := range f.RoleTypes {
				f.RoleTypes[j] = RoleType{Name: d.str(), SourceFile: d.str()}
			}
		}
		if n := d.int(); n > 0 {
			if d.err != nil || n > len(data) {
				return "", nil, errBadCache
			}
			f.Defs = make([]SearchResult, n)
			for j := range f.Defs {
				f.Defs[j] = SearchResult{Name: d.str(), Kind: d.str(), SourceFile: d.str(), Line: d.int()}
			}
		}
		if n := d.int(); n > 0 {
			if d.err != nil || n > len(data) {
				return "", nil, errBadCache
			}
			f.Candidates = make([]VarRoleInfo, n)
			for j := range f.Candidates {
				f.Candidates[j] = VarRoleInfo{VarName: d.str(), RoleType: d.str(), SourceFile: d.str()}
			}
		}
		if n := d.int(); n > 0 {
			if d.err != nil || n > len(data) {
				return "", nil, errBadCache
			}
			f.Literals = make([]LiteralField, n)
			for j := range f.Literals {
				f.Literals[j] = LiteralField{
					DefName:    d.str(),
					TypeName:   d.str(),
					FieldName:  d.str(),
					FieldValue: d.str(),
					SourceFile: d.str(),
					Line:       d.int(),
				}
			}
		}
		if n := d.int(); n > 0 {
			if d.err != nil || n > len(data) {
				return "", nil, errBadCache
			}
			f.Pragmas = make([]Pragma, n)
			for j := range f.Pragmas {
				f.Pragmas[j] = Pragma{
					DefName:    d.str(),
					SourceFile: d.str(),
					Line:       d.int(),
					Key:        d.str(),
					Value:      d.str(),
				}
			}
		}
		if d.err != nil {
			return "", nil, d.err
		}
		frags[path] = f
	}
	if d.err != nil {
		return "", nil, d.err
	}
	return dir, frags, nil
}

// unsafeString aliases a byte slice as a string without copying. Safe here
// because `data` is read-only for the decoder's lifetime and never handed back
// to a caller who could mutate it; it is the difference between one allocation
// and sixty thousand.
func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}
