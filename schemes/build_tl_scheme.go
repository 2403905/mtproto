package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type nametype struct {
	name  string
	_type string
	ttype string
	flag  int
}

type constuctor struct {
	id        string
	predicate string
	params    []nametype
	_type     string
}

func normalize(s string) string {
	x := []byte(s)
	for i, r := range x {
		if r == '.' {
			x[i] = '_'
		}
	}
	y := string(x)
	if y == "type" {
		return "_type"
	}
	return y
}

func lower_first(s string) string {
	return strings.ToLower(string(s[0])) + string(s[1:len(s)])
}

func main() {
	var err error
	var parsed interface{}

	// read json file from stdin
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		return
	}

	// parse json
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	err = d.Decode(&parsed)
	if err != nil {
		fmt.Println(err)
		return
	}

	// process constructors
	_order := make([]string, 0, 1000)
	_cons := make(map[string]constuctor, 1000)
	var _types []string

	parsefunc := func(data []interface{}, kind string) {
		for _, data := range data {
			data := data.(map[string]interface{})

			// id
			idx, err := strconv.Atoi(data["id"].(string))
			if err != nil {
				fmt.Println(err)
				return
			}
			_id := fmt.Sprintf("0x%08x", uint32(idx))

			// predicate
			_predicate := normalize(data[kind].(string))

			if _predicate == "vector" {
				continue
			}

			// params
			_params := make([]nametype, 0, 16)
			params := data["params"].([]interface{})
			for _, params := range params {
				params := params.(map[string]interface{})
				_name := normalize(params["name"].(string))
				_type := normalize(params["type"].(string))
				_ttype := ""
				_flag := -1

				flagRegex := regexp.MustCompile("([a-zA-Z]+)_(\\d+)\\?([a-zA-Z<>]+)")
				m := flagRegex.FindStringSubmatch(_type)
				if len(m) > 0 {
					_type = m[3]
					_flag, _ = strconv.Atoi(m[2])
				}

				_params = append(_params, nametype{_name, _type, _ttype, _flag})
			}

			// type
			_type := normalize(data["type"].(string))

			_order = append(_order, _predicate)
			_cons[_predicate] = constuctor{_id, _predicate, _params, _type}
			if kind == "predicate" {
				sort.Strings(_types)
				i := sort.SearchStrings(_types, _type)
				if i >= len(_types) || _types[i] != _type {
					_types = append(_types, _predicate)
				}
			}
		}
	}
	parsefunc(parsed.(map[string]interface{})["constructors"].([]interface{}), "predicate")
	parsefunc(parsed.(map[string]interface{})["methods"].([]interface{}), "method")

	// constants
	fmt.Print(`package mtproto
import (
	"fmt"
	"encoding/binary"
	"errors"
)
const (
`)
	for _, key := range _order {
		c := _cons[key]
		fmt.Printf("crc_%s = %s\n", c.predicate, c.id)
	}
	fmt.Print(")\n\n")

	// type structs
	for _, key := range _order {
		c := _cons[key]
		fmt.Printf("type TL_%s struct {\n", c.predicate)
		for _i, t := range c.params {
			fmt.Printf("%s\t", t.name)
			switch t._type {
			case "#":
				fmt.Printf("uint32")
			case "int":
				fmt.Printf("int32")
			case "long":
				fmt.Printf("int64")
			case "string":
				fmt.Printf("string")
			case "double":
				fmt.Printf("float64")
			case "bytes":
				fmt.Printf("[]byte")
			case "Bool", "true":
				fmt.Printf("bool")
			case "Vector<int>":
				fmt.Printf("[]int32")
			case "Vector<long>":
				fmt.Printf("[]int64")
			case "Vector<string>":
				fmt.Printf("[]string")
			case "Vector<double>":
				fmt.Printf("[]float64")
			case "!X":
				fmt.Printf("TL")
			default:
				var inner string
				var k string

				n, _ := fmt.Sscanf(t._type, "Vector<%s", &inner)
				if n == 1 {
					k = inner[:len(inner)-1]
				} else {
					k = t._type
				}

				lk := lower_first(k)

				i := sort.SearchStrings(_types, lk)
				if i < len(_types) && _types[i] == lk {
					if n == 1 {
						fmt.Printf("[]TL_%s", lk)
					} else {
						c.params[_i].ttype = fmt.Sprintf("TL_%s", lk)
						fmt.Printf(c.params[_i].ttype)
					}
				} else {
					if n == 1 {
						fmt.Printf("[]TL // %s", k)
					} else {
						fmt.Printf("TL // %s", k)
					}
				}
			}
			fmt.Printf("\n")
		}
		fmt.Printf("}\n\n")
	}

	// encode funcs
	for _, key := range _order {
		c := _cons[key]
		fmt.Printf("func (e TL_%s) encode() []byte {\n", c.predicate)
		fmt.Printf("x := NewEncodeBuf(512)\n")
		fmt.Printf("x.UInt(crc_%s)\n", c.predicate)
		for _, t := range c.params {
			if t.flag > -1 {
				fmt.Printf("if (e.flags & (1 << %d)) > 0 {", t.flag)
			}
			switch t._type {
			case "int":
				fmt.Printf("x.Int(e.%s)\n", t.name)
			case "#":
				fmt.Printf("x.UInt(e.%s)\n", t.name)
			case "Bool":
				fmt.Printf("x.Bool(e.%s)\n", t.name)
			case "true":
				// nothing
			case "long":
				fmt.Printf("x.Long(e.%s)\n", t.name)
			case "double":
				fmt.Printf("x.Double(e.%s)\n", t.name)
			case "string":
				fmt.Printf("x.String(e.%s)\n", t.name)
			case "Vector<int>":
				fmt.Printf("x.VectorInt(e.%s)\n", t.name)
			case "Vector<long>":
				fmt.Printf("x.VectorLong(e.%s)\n", t.name)
			case "bytes":
				fmt.Printf("x.StringBytes(e.%s)\n", t.name)
			case "Vector<string>":
				fmt.Printf("x.VectorString(e.%s)\n", t.name)
			case "!X":
				fmt.Printf("x.Bytes(e.%s.encode())\n", t.name)
			case "Vector<double>":
				panic(fmt.Sprintf("Unsupported %s", t._type))
			default:
				var inner string
				n, _ := fmt.Sscanf(t._type, "Vector<%s", &inner)
				if n == 1 {
					lk := lower_first(inner[:len(inner)-1])
					i := sort.SearchStrings(_types, lk)
					if i < len(_types) && _types[i] == lk {
						fmt.Printf("x.Vector_%s(e.%s)\n", lk, t.name)
					} else {
						fmt.Printf("x.Vector(e.%s)\n", t.name)
					}
				} else {
					fmt.Printf("x.Bytes(e.%s.encode())\n", t.name)
				}
			}
			if t.flag > -1 {
				fmt.Print("}\n")
			}
		}
		fmt.Printf("return x.buf\n")
		fmt.Printf("}\n\n")

	}
	vencode := `
func (e *EncodeBuf) Vector_%s(v []TL_%s) {
	x := make([]byte, 512)
	binary.LittleEndian.PutUint32(x, crc_vector)
	binary.LittleEndian.PutUint32(x[4:], uint32(len(v)))
	e.buf = append(e.buf, x...)
	for _, v := range v {
		e.buf = append(e.buf, v.encode()...)
	}
}
`
	vdecode := `
func (db *DecodeBuf) Vector_%s() []TL_%s {
	constructor := db.UInt()
	if db.err != nil {
		return nil
	}
	if constructor != crc_vector {
		db.err = fmt.Errorf("DecodeVector: Wrong constructor (0x%%08x)", constructor)
		return nil
	}
	size := db.Int()
	if db.err != nil {
		return nil
	}
	if size < 0 {
		db.err = errors.New("DecodeVector: Wrong size")
		return nil
	}
	x := make([]TL_%s, size)
	i := int32(0)
	for i < size {
		y := db.Object().(TL_%s)
		if db.err != nil {
			return nil
		}
		x[i] = y
		i++
	}
	return x
}
`
	// decode & encode vectors
	var vectors []string

	for _, key := range _order {
		c := _cons[key]
		for _, t := range c.params {
			var inner string

			n, _ := fmt.Sscanf(t._type, "Vector<%s", &inner)
			if n != 1 {
				continue
			}

			lk := lower_first(inner[:len(inner)-1])

			q := sort.SearchStrings(vectors, lk)
			i := sort.SearchStrings(_types, lk)
			if i < len(_types) && _types[i] == lk && n == 1 {
				if q >= len(vectors) || vectors[q] != lk {
					fmt.Printf(vdecode, lk, lk, lk, lk)
					fmt.Printf(vencode, lk, lk)
					vectors = append(vectors, lk)
					sort.Strings(vectors)
				}
			}
		}
	}

	// decode funcs
	fmt.Println(`
func (m *DecodeBuf) ObjectGenerated(constructor uint32) (r TL) {
	switch constructor {`)

	for _, key := range _order {
		c := _cons[key]
		var flag bool
		begin := ""
		endin := ",\n"
		fmt.Printf("case crc_%s:\n", c.predicate)
		if len(c.params) > 0 && c.params[0]._type == "#" {
			flag = true
			endin = "\n"
			fmt.Printf("rr := TL_%s{}\n", c.predicate)
		} else {
			fmt.Printf("r = TL_%s{\n", c.predicate)
		}

		for _, t := range c.params {
			if flag {
				begin = fmt.Sprintf("rr.%s = ", t.name)
			}
			if t.flag > -1 {
				fmt.Printf("if (rr.flags & (1 << %d)) > 0 {", t.flag)
			}

			switch t._type {
			case "int":
				fmt.Printf("%sm.Int()%s", begin, endin)
			case "#":
				fmt.Printf("%sm.UInt()%s", begin, endin)
			case "Bool":
				fmt.Printf("%sm.Bool()%s", begin, endin)
			case "long":
				fmt.Printf("%sm.Long()%s", begin, endin)
			case "double":
				fmt.Printf("%sm.Double()%s", begin, endin)
			case "string":
				fmt.Printf("%sm.String()%s", begin, endin)
			case "Vector<int>":
				fmt.Printf("%sm.VectorInt()%s", begin, endin)
			case "Vector<long>":
				fmt.Printf("%sm.VectorLong()%s", begin, endin)
			case "bytes":
				fmt.Printf("%sm.StringBytes()%s", begin, endin)
			case "Vector<string>":
				fmt.Printf("%sm.VectorString()%s", begin, endin)
			case "!X":
				fmt.Printf("%sm.Object()%s", begin, endin)
			case "true":
				fmt.Printf("%strue", begin)
			case "Vector<double>":
				panic(fmt.Sprintf("Unsupported %s", t._type))
			default:
				var inner string
				n, _ := fmt.Sscanf(t._type, "Vector<%s", &inner)
				if n == 1 {
					lk := lower_first(inner[:len(inner)-1])
					i := sort.SearchStrings(_types, lk)
					if i < len(_types) && _types[i] == lk {
						fmt.Printf("%sm.Vector_%s()%s", begin, lk, endin)
					} else {
						fmt.Printf("%sm.Vector()%s", begin, endin)
					}
				} else {
					if t.ttype != "" {
						fmt.Printf("%sm.Object().(%s)%s", begin, t.ttype, endin)
					} else {
						fmt.Printf("%sm.Object()%s", begin, endin)
					}
				}
			}

			if t.flag > -1 {
				fmt.Print("}\n")
			}
		}
		if flag {
			fmt.Print("r = rr\n\n")
		} else {
			fmt.Print("}\n\n")
		}
	}

	fmt.Println(`
	default:
		m.err = fmt.Errorf("Unknown constructor: %08x", constructor)
		return nil

	}

	if m.err != nil {
		return nil
	}

	return
}`)
}
