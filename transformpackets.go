// Author  Raido Pahtma
// License MIT

package sfconnection

import "fmt"
import "reflect"
import "strings"
import "strconv"
import "regexp"
import "encoding/binary"
import "bytes"
import "errors"

func SerializePacket(m interface{}) []byte {
	buf := new(bytes.Buffer)
	transformPacket(true, buf, m)
	return buf.Bytes()
}

func DeserializePacket(m interface{}, data []byte) error {
	buf := bytes.NewBuffer(data)
	return transformPacket(false, buf, m)
}

func parseSfPacketTag(tag string) map[string]string {
	mp := make(map[string]string)
	if idx := strings.Index(tag, ":"); idx != -1 {
		if tag[:idx] == "sfpacket" {
			val, err := strconv.Unquote(tag[idx+1:])
			if err != nil {
				panic(fmt.Sprintf("Bad tag %s", tag))
			}
			re := regexp.MustCompile("len\\(([a-zA-Z0-9.-]+)\\)")
			match := re.FindStringSubmatch(val)
			if len(match) == 2 {
				mp["len"] = match[1]
			}
		}
	}
	return mp
}

func readUintFrom(kind reflect.Kind, buf *bytes.Buffer) (uint64, error) {
	switch kind {
	case reflect.Uint8:
		var v uint8
		err := binary.Read(buf, binary.BigEndian, &v)
		return uint64(v), err
	case reflect.Uint16:
		var v uint16
		err := binary.Read(buf, binary.BigEndian, &v)
		return uint64(v), err
	case reflect.Uint32:
		var v uint32
		err := binary.Read(buf, binary.BigEndian, &v)
		return uint64(v), err
	case reflect.Uint64:
		var v uint64
		err := binary.Read(buf, binary.BigEndian, &v)
		return v, err
	}
	panic(fmt.Sprintf("%s not supported for deserialization!", kind))
}

func readIntFrom(kind reflect.Kind, buf *bytes.Buffer) (int64, error) {
	switch kind {
	case reflect.Int8:
		var v int8
		err := binary.Read(buf, binary.BigEndian, &v)
		return int64(v), err
	case reflect.Int16:
		var v int16
		err := binary.Read(buf, binary.BigEndian, &v)
		return int64(v), err
	case reflect.Int32:
		var v int32
		err := binary.Read(buf, binary.BigEndian, &v)
		return int64(v), err
	case reflect.Int64:
		var v int64
		err := binary.Read(buf, binary.BigEndian, &v)
		return v, err
	}
	panic(fmt.Sprintf("%s not supported for deserialization!", kind))
}

func transformPacket(serialize bool, buf *bytes.Buffer, m interface{}) error {
	val := reflect.ValueOf(m)      // reflect.Value
	if val.Kind() == reflect.Ptr { // get the actual type from the pointer
		val = val.Elem()
	} else {
		panic("Transformation is only possible, if a pointer is passed!") // This applies in both directions, serialization could work in some cases without pointers as well
	}

	if val.Type().Kind() != reflect.Struct { // Only structs supported
		panic(fmt.Sprintf("%v type can't be serialized!", val.Type().Kind()))
	}

	arrlenmap := make(map[string]string) // Map containing names of length fields for variable-length fields

	for i := 0; i < val.NumField(); i++ {
		buflen := buf.Len()           // Even unsuccessful reads consume data from the buffer, so keep this for error messages
		sfield := val.Type().Field(i) // StructField
		if sfield.Anonymous {
			panic(fmt.Sprintf("Cannot serialize anonymous fields!"))
		}

		if len(sfield.Tag) > 0 {
			tgs := parseSfPacketTag(string(sfield.Tag))
			arrname, ok := tgs["len"]
			if ok {
				arr := val.FieldByName(arrname)
				if !arr.IsValid() {
					panic(fmt.Sprintf("Cannot find member %s for length tag %s!", arrname, sfield.Name))
				}
				val.Field(i).SetUint(uint64(arr.Len())) // Set the value to the length of the actual object
				arrlenmap[arrname] = sfield.Name
			}
		}

		switch sfield.Type.Kind() {
		case reflect.Array:
			if serialize {
				if err := binary.Write(buf, binary.BigEndian, val.Field(i).Interface()); err != nil {
					panic(err)
				}
			} else { // Only works for bytes(and uint8)
				var b []byte = make([]byte, val.Field(i).Len())
				if n, err := buf.Read(b); err == nil && n == val.Field(i).Len() {
					reflect.Copy(val.Field(i), reflect.ValueOf(b))
				} else {
					return errors.New(fmt.Sprintf("Unable to deserialize field %s (err=%v had %d bytes, needed %d)", sfield.Name, err, buflen, val.Field(i).Len()))
				}
			}

		case reflect.Bool:
			if serialize {
				var boolval uint8
				if val.Field(i).Bool() {
					boolval = 1
				}
				if err := binary.Write(buf, binary.BigEndian, boolval); err != nil {
					panic(err)
				}
			} else {
				var v uint8
				if err := binary.Read(buf, binary.BigEndian, &v); err == nil {
					val.Field(i).SetBool(v != 0)
				} else {
					return errors.New(fmt.Sprintf("Unable to deserialize field %s (had %d bytes)", sfield.Name, buflen))
				}
			}

		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if serialize {
				if err := binary.Write(buf, binary.BigEndian, val.Field(i).Interface()); err != nil {
					panic(err)
				}
			} else {
				if v, err := readUintFrom(sfield.Type.Kind(), buf); err == nil {
					val.Field(i).SetUint(v)
				} else {
					return errors.New(fmt.Sprintf("Unable to deserialize field %s (had %d bytes)", sfield.Name, buflen))
				}
			}

		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if serialize {
				if err := binary.Write(buf, binary.BigEndian, val.Field(i).Interface()); err != nil {
					panic(err)
				}
			} else {
				if v, err := readIntFrom(sfield.Type.Kind(), buf); err == nil {
					val.Field(i).SetInt(v)
				} else {
					return errors.New(fmt.Sprintf("Unable to deserialize field %s (had %d bytes)", sfield.Name, buflen))
				}
			}

		case reflect.Slice:
			if sfield.Type == reflect.TypeOf([]byte(nil)) { // Seems to recognize uint8 as well
				var slicelen int
				if lenname, ok := arrlenmap[sfield.Name]; ok {
					length := val.FieldByName(lenname)
					if !length.IsValid() {
						panic(fmt.Sprintf("Cannot find length for member %s!", sfield.Name))
					}
					slicelen = int(length.Uint()) // Has a length parameter specified
				} else if i == val.NumField()-1 {
					slicelen = buf.Len() // Last one is also ok
				} else {
					panic(fmt.Sprintf("Processing of field %s(%s) is not possible, no length field specified!", sfield.Name, sfield.Type))
				}

				if serialize {
					if _, err := buf.Write(val.Field(i).Bytes()); err != nil {
						panic(fmt.Sprintf("Serialization of field %s(%s) failed: %s", sfield.Name, sfield.Type, err))
					}
				} else {
					var b []byte = make([]byte, slicelen)
					if n, err := buf.Read(b); err == nil && n == slicelen {
						val.Field(i).SetBytes(b)
					} else {
						return errors.New(fmt.Sprintf("Unable to deserialize field %s (err=%v had %d bytes, needed %d)", sfield.Name, err, buflen, slicelen))
					}
				}
			} else {
				panic(fmt.Sprintf("Serialization of field %s(%s) is not (yet?) supported!", sfield.Name, sfield.Type))
			}

		case reflect.String:
			var slicelen int
			if lenname, ok := arrlenmap[sfield.Name]; ok {
				length := val.FieldByName(lenname)
				if !length.IsValid() {
					panic(fmt.Sprintf("Cannot find length for member %s!", sfield.Name))
				}
				slicelen = int(length.Uint()) // Has a length parameter specified
			} else if i == val.NumField()-1 {
				slicelen = buf.Len() // Last one is also ok
			} else {
				panic(fmt.Sprintf("Serialization of field %s(%s) is not possible, no length field specified!", sfield.Name, sfield.Type))
			}

			if serialize {
				if err := binary.Write(buf, binary.BigEndian, []byte(val.Field(i).String())); err != nil {
					panic(fmt.Sprintf("Serialization of field %s(%s) failed: %s", sfield.Name, sfield.Type, err))
				}
			} else {
				var b []byte = make([]byte, slicelen)
				if n, err := buf.Read(b); err == nil && n == slicelen {
					val.Field(i).SetString(string(b))
				} else {
					return errors.New(fmt.Sprintf("Unable to deserialize field %s (err=%v had %d bytes, needed %d)", sfield.Name, err, buflen, slicelen))
				}
			}
		case reflect.Struct:
			panic(fmt.Sprintf("Serialization of field structs %s(%s) is not yet supported!", sfield.Name, sfield.Type))
		default:
			panic(fmt.Sprintf("Serialization of field %s(%s) is not (yet?) supported!", sfield.Name, sfield.Type))
		}
		//fmt.Printf("%s %s %s %v\n", sfield.Name, sfield.Type, sfield.Tag, arrlenmap)
	}

	if !serialize && buf.Len() != 0 {
		return errors.New(fmt.Sprintf("Buffer contains %d extra bytes!", buf.Len()))
	}

	return nil
}
