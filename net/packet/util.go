package packet

import (
	"errors"
	"io"
	"reflect"
)

type Integers interface {
	VarInt | VarLong | Int | Long | Byte | UnsignedByte | Short | UnsignedShort
}

// Ary is used to send or receive the packet field like "Array of X"
// which has a count must be known from the context.
//
// Typically, you must decode an integer representing the length. Then
// receive the corresponding amount of data according to the length.
// In this case, the field Len should be a pointer of integer type so
// the value can be updating when Packet.Scan() method is decoding the
// previous field.
// In some special cases, you might want to read an "Array of X" with a fix length.
// So it's allowed to directly set an integer type Len, but not a pointer.
//
// Note that Ary now handle the prefixed Length field.
type Ary[L Integers, A any] []A

func (a Ary[L, A]) WriteTo(r io.Writer) (n int64, err error) {
	n, err = any(L(len(a))).(FieldEncoder).WriteTo(r)
	if err != nil {
		return
	}
	for _, v := range a {
		nn, err := any(v).(FieldEncoder).WriteTo(r)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (a *Ary[L, A]) ReadFrom(r io.Reader) (n int64, err error) {
	var length L
	n, err = any(&length).(FieldDecoder).ReadFrom(r)
	if err != nil {
		return
	}
	*a = make([]A, length)
	for i := range *a {
		nn, err := any(&((*a)[i])).(FieldDecoder).ReadFrom(r)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, err
}

type Opt struct {
	Has   interface{} // Pointer of bool, or `func() bool`
	Field interface{} // FieldEncoder, FieldDecoder or both (Field)
}

func (o Opt) has() bool {
	v := reflect.ValueOf(o.Has)
	for {
		switch v.Kind() {
		case reflect.Ptr:
			v = v.Elem()
		case reflect.Bool:
			return v.Bool()
		case reflect.Func:
			return v.Interface().(func() bool)()
		default:
			panic(errors.New("unsupported Has value"))
		}
	}
}

func (o Opt) WriteTo(w io.Writer) (int64, error) {
	if o.has() {
		return o.Field.(FieldEncoder).WriteTo(w)
	}
	return 0, nil
}

func (o Opt) ReadFrom(r io.Reader) (int64, error) {
	if o.has() {
		return o.Field.(FieldDecoder).ReadFrom(r)
	}
	return 0, nil
}

type Tuple []interface{} // FieldEncoder, FieldDecoder or both (Field)

// WriteTo write Tuple to io.Writer, panic when any of filed don't implement FieldEncoder
func (t Tuple) WriteTo(w io.Writer) (n int64, err error) {
	for _, v := range t {
		nn, err := v.(FieldEncoder).WriteTo(w)
		if err != nil {
			return n, err
		}
		n += nn
	}
	return
}

// ReadFrom read Tuple from io.Reader, panic when any of field don't implement FieldDecoder
func (t Tuple) ReadFrom(r io.Reader) (n int64, err error) {
	for _, v := range t {
		nn, err := v.(FieldDecoder).ReadFrom(r)
		if err != nil {
			return n, err
		}
		n += nn
	}
	return
}
