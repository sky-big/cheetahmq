// Copyright (c) 2012, Sean Treadway, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Source code and contact info at http://github.com/streadway/amqp

// +build ignore

package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

var (
	ErrUnknownType   = errors.New("Unknown field type in gen")
	ErrUnknownDomain = errors.New("Unknown domain type in gen")
)

var amqpTypeToNative = map[string]string{
	"bit":        "bool",
	"octet":      "byte",
	"shortshort": "uint8",
	"short":      "uint16",
	"long":       "uint32",
	"longlong":   "uint64",
	"timestamp":  "time.Time",
	"table":      "Table",
	"shortstr":   "string",
	"longstr":    "string",
}

type Rule struct {
	Name string   `xml:"name,attr"`
	Docs []string `xml:"doc"`
}

type Doc struct {
	Type string `xml:"type,attr"`
	Body string `xml:",innerxml"`
}

type Chassis struct {
	Name      string `xml:"name,attr"`
	Implement string `xml:"implement,attr"`
}

type Assert struct {
	Check  string `xml:"check,attr"`
	Value  string `xml:"value,attr"`
	Method string `xml:"method,attr"`
}

type Field struct {
	Name     string   `xml:"name,attr"`
	Domain   string   `xml:"domain,attr"`
	Type     string   `xml:"type,attr"`
	Label    string   `xml:"label,attr"`
	Reserved bool     `xml:"reserved,attr"`
	Docs     []Doc    `xml:"doc"`
	Asserts  []Assert `xml:"assert"`
}

type Response struct {
	Name string `xml:"name,attr"`
}

type Method struct {
	Name        string    `xml:"name,attr"`
	Response    Response  `xml:"response"`
	Synchronous bool      `xml:"synchronous,attr"`
	Content     bool      `xml:"content,attr"`
	Index       string    `xml:"index,attr"`
	Label       string    `xml:"label,attr"`
	Docs        []Doc     `xml:"doc"`
	Rules       []Rule    `xml:"rule"`
	Fields      []Field   `xml:"field"`
	Chassis     []Chassis `xml:"chassis"`
}

type Class struct {
	Name    string    `xml:"name,attr"`
	Handler string    `xml:"handler,attr"`
	Index   string    `xml:"index,attr"`
	Label   string    `xml:"label,attr"`
	Docs    []Doc     `xml:"doc"`
	Methods []Method  `xml:"method"`
	Chassis []Chassis `xml:"chassis"`
}

type Domain struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Label string `xml:"label,attr"`
	Rules []Rule `xml:"rule"`
	Docs  []Doc  `xml:"doc"`
}

type Constant struct {
	Name  string `xml:"name,attr"`
	Value int    `xml:"value,attr"`
	Class string `xml:"class,attr"`
	Doc   string `xml:"doc"`
}

type Amqp struct {
	Major   int    `xml:"major,attr"`
	Minor   int    `xml:"minor,attr"`
	Port    int    `xml:"port,attr"`
	Comment string `xml:"comment,attr"`

	Constants []Constant `xml:"constant"`
	Domains   []Domain   `xml:"domain"`
	Classes   []Class    `xml:"class"`
}

type renderer struct {
	Root       Amqp
	bitcounter int
}

type fieldset struct {
	AmqpType   string
	NativeType string
	Fields     []Field
	*renderer
}

var (
	helpers = template.FuncMap{
		"public":  public,
		"private": private,
		"clean":   clean,
	}

	packageTemplate = template.Must(template.New("package").Funcs(helpers).Parse(`
	// Copyright (c) 2012, Sean Treadway, SoundCloud Ltd.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.

    /* GENERATED FILE - DO NOT EDIT */
    /* Rebuild from the spec/gen.go tool */

    {{with .Root}}
    package main

    import (
        "fmt"
        "encoding/binary"
        "io"
        "os"
        "log"
    )

	// Error codes that can be sent from the server during a connection or
	// channel exception or used by the client to indicate a class of error like
	// ErrCredentials.  The text of the error is likely more interesting than
	// these constants.
	const (
	   {{range $c := .Constants}}
	       {{if $c.IsError}}{{.Name | public}}{{else}}{{.Name | private}}{{end}} = {{.Value}}{{end}}
    )

	func isSoftExceptionCode(code int) bool {
		switch code {
		{{range $c := .Constants}} {{if $c.IsSoftError}} case {{$c.Value}}:
			return true
		{{end}}{{end}}
		}
		return false
	}

    {{range .Classes}}
        {{$class := .}}
            {{range .Methods}}
                {{$method := .}}
    			     {{$struct := $.StructName $class.Name $method.Name}}
                    {{if .Docs}}/* {{range .Docs}} {{.Body | clean}} {{end}} */{{end}}
                    type {{$struct}} struct {
                        {{range .Fields}}
                            {{$.FieldName .}} {{$.FieldType . | $.NativeType}} {{if .Label}}// {{.Label}}{{end}}{{end}}
        				{{if .Content}}Properties properties
        				Body []byte{{end}}
                    }

        			func (me *{{$struct}}) id() (uint16, uint16) {
        				return {{$class.Index}}, {{$method.Index}}
        			}

        			func (me *{{$struct}}) wait() (bool) {
        				return {{.Synchronous}}{{if $.HasField "NoWait" .}} && !me.NoWait{{end}}
        			}

        			{{if .Content}}
                        func (me *{{$struct}}) getContent() (properties, []byte) {
                            return me.Properties, me.Body
                        }

                        func (me *{{$struct}}) setContent(props properties, body []byte) {
                            me.Properties, me.Body = props, body
                        }

                        func (me *{{$struct}}) isHaveContent() bool {
                            return true
                        }
                    {{else}}
                        func (me *{{$struct}}) isHaveContent() bool {
                            return false
                        }
        			{{end}}
                    func (me *{{$struct}}) write(w io.Writer) (err error) {
            			{{if $.HasType "bit" $method}}var bits byte{{end}}
                        {{.Fields | $.Fieldsets | $.Partial "enc-"}}
                        return
                    }

                    func (me *{{$struct}}) read(r io.Reader) (err error) {
        				{{if $.HasType "bit" $method}}var bits byte{{end}}
                            {{.Fields | $.Fieldsets | $.Partial "dec-"}}
                        return
                    }
            {{end}}
    {{end}}

    func (me *Reader) parseMethodFrame(channel uint16, size uint32) (f frame, err error) {
        mf := &methodFrame {
            ChannelId: channel,
        }

        if err = binary.Read(me.R, binary.BigEndian, &mf.ClassId); err != nil {
          return
        }

        if err = binary.Read(me.R, binary.BigEndian, &mf.MethodId); err != nil {
          return
        }

        switch mf.ClassId {
            {{range .Classes}}
            {{$class := .}}
                case {{.Index}}: // {{.Name}}
                    switch mf.MethodId {
                        {{range .Methods}}
                            case {{.Index}}: // {{$class.Name}} {{.Name}}
                            //fmt.Println("NextMethod: class:{{$class.Index}} method:{{.Index}}")
                            method := &{{$.StructName $class.Name .Name}}{}
                            if err = method.read(me.R); err != nil {
                              return
                            }
                            mf.Method = method
                            log.Printf("<<<<<<<<<< receieve method : {{$class.Name}} {{.Name}} class:{{$class.Index}} method:{{.Index}} : %v\n", method)
                        {{end}}
                      default:
                        return nil, fmt.Errorf("Bad method frame, unknown method %d for class %d", mf.MethodId, mf.ClassId)
                    }
            {{end}}
            default:
              return nil, fmt.Errorf("Bad method frame, unknown class %d", mf.ClassId)
            }

            return mf, nil
        }
            {{end}}

  {{define "enc-bit"}}
    {{range $off, $field := .Fields}}
    if me.{{$field | $.FieldName}} { bits |= 1 << {{$off}} }
    {{end}}
    if err = binary.Write(w, binary.BigEndian, bits); err != nil { return }
  {{end}}

  {{define "enc-octet"}}
    {{range .Fields}} if err = binary.Write(w, binary.BigEndian, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-shortshort"}}
    {{range .Fields}} if err = binary.Write(w, binary.BigEndian, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-short"}}
    {{range .Fields}} if err = binary.Write(w, binary.BigEndian, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-long"}}
    {{range .Fields}} if err = binary.Write(w, binary.BigEndian, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-longlong"}}
    {{range .Fields}} if err = binary.Write(w, binary.BigEndian, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-timestamp"}}
    {{range .Fields}} if err = writeTimestamp(w, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-shortstr"}}
    {{range .Fields}} if err = writeShortstr(w, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-longstr"}}
    {{range .Fields}} if err = writeLongstr(w, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "enc-table"}}
    {{range .Fields}} if err = writeTable(w, me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-bit"}}
    if err = binary.Read(r, binary.BigEndian, &bits); err != nil {
      return
    }
    {{range $off, $field := .Fields}} me.{{$field | $.FieldName}} = (bits & (1 << {{$off}}) > 0)
    {{end}}
  {{end}}

  {{define "dec-octet"}}
    {{range .Fields}} if err = binary.Read(r, binary.BigEndian, &me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-shortshort"}}
    {{range .Fields}} if err = binary.Read(r, binary.BigEndian, &me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-short"}}
    {{range .Fields}} if err = binary.Read(r, binary.BigEndian, &me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-long"}}
    {{range .Fields}} if err = binary.Read(r, binary.BigEndian, &me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-longlong"}}
    {{range .Fields}} if err = binary.Read(r, binary.BigEndian, &me.{{. | $.FieldName}}); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-timestamp"}}
    {{range .Fields}} if me.{{. | $.FieldName}}, err = readTimestamp(r); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-shortstr"}}
    {{range .Fields}} if me.{{. | $.FieldName}}, err = readShortstr(r); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-longstr"}}
    {{range .Fields}} if me.{{. | $.FieldName}}, err = readLongstr(r); err != nil { return }
    {{end}}
  {{end}}

  {{define "dec-table"}}
    {{range .Fields}} if me.{{. | $.FieldName}}, err = readTable(r); err != nil { return }
    {{end}}
  {{end}}
  // get amqp error info
  func GetAmqpErrorInfo(reason string) (bool, uint16, string) {
    switch reason {
    case "content_too_large":
        return false, ContentTooLarge, "CONTENT_TOO_LARGE"
    case "no_route":
        return false, NoRoute, "NO_ROUTE"
    case "no_consumers":
        return false, NoConsumers, "NO_CONSUMERS"
    case "access_refused":
        return false, AccessRefused, "ACCESS_REFUSED"
    case "not_found":
        return false, NotFound, "NOT_FOUND"
    case "resource_locked":
        return false, ResourceLocked, "RESOURCE_LOCKED"
    case "precondition_failed":
        return false, PreconditionFailed, "PRECONDITION_FAILED"
    case "connection_forced":
        return true, ConnectionForced, "CONNECTION_FORCED"
    case "invalid_path":
        return true, InvalidPath, "INVALID_PATH"
    case "frame_error":
        return true, FrameError, "FRAME_ERROR"
    case "syntax_error":
        return true, SyntaxError, "SYNTAX_ERROR"
    case "command_invalid":
        return true, CommandInvalid, "COMMAND_INVALID"
    case "channel_error":
        return true, ChannelError, "CHANNEL_ERROR"
    case "unexpected_frame":
        return true, UnexpectedFrame, "UNEXPECTED_FRAME"
    case "resource_error":
        return true, ResourceError, "RESOURCE_ERROR"
    case "not_allowed":
        return true, NotAllowed, "NOT_ALLOWED"
    case "not_implemented":
        return true, NotImplemented, "NOT_IMPLEMENTED"
    case "internal_error":
        return true, InternalError, "INTERNAL_ERROR"
    }
    os.Exit(1)
    return false, 0, ""
  }
  `))
)

func (me *Constant) IsError() bool {
	return strings.Contains(me.Class, "error")
}

func (me *Constant) IsSoftError() bool {
	return me.Class == "soft-error"
}

func (me *renderer) Partial(prefix string, fields []fieldset) (s string, err error) {
	var buf bytes.Buffer
	for _, set := range fields {
		name := prefix + set.AmqpType
		t := packageTemplate.Lookup(name)
		if t == nil {
			return "", errors.New(fmt.Sprintf("Missing template: %s", name))
		}
		if err = t.Execute(&buf, set); err != nil {
			return
		}
	}
	return string(buf.Bytes()), nil
}

// Groups the fields so that the right encoder/decoder can be called
func (me *renderer) Fieldsets(fields []Field) (f []fieldset, err error) {
	if len(fields) > 0 {
		for _, field := range fields {
			cur := fieldset{}
			cur.AmqpType, err = me.FieldType(field)
			if err != nil {
				return
			}

			cur.NativeType, err = me.NativeType(cur.AmqpType)
			if err != nil {
				return
			}
			cur.Fields = append(cur.Fields, field)
			f = append(f, cur)
		}

		i, j := 0, 1
		for j < len(f) {
			if f[i].AmqpType == f[j].AmqpType {
				f[i].Fields = append(f[i].Fields, f[j].Fields...)
			} else {
				i++
				f[i] = f[j]
			}
			j++
		}
		return f[:i+1], nil
	}

	return
}

func (me *renderer) HasType(typ string, method Method) bool {
	for _, f := range method.Fields {
		name, _ := me.FieldType(f)
		if name == typ {
			return true
		}
	}
	return false
}

func (me *renderer) HasField(field string, method Method) bool {
	for _, f := range method.Fields {
		name := me.FieldName(f)
		if name == field {
			return true
		}
	}
	return false
}

func (me *renderer) Domain(field Field) (domain Domain, err error) {
	for _, domain = range me.Root.Domains {
		if field.Domain == domain.Name {
			return
		}
	}
	return domain, nil
	//return domain, ErrUnknownDomain
}

func (me *renderer) FieldName(field Field) (t string) {
	t = public(field.Name)

	if field.Reserved {
		t = strings.ToLower(t)
	}

	return
}

func (me *renderer) FieldType(field Field) (t string, err error) {
	t = field.Type

	if t == "" {
		var domain Domain
		domain, err = me.Domain(field)
		if err != nil {
			return "", err
		}
		t = domain.Type
	}

	return
}

func (me *renderer) NativeType(amqpType string) (t string, err error) {
	if t, ok := amqpTypeToNative[amqpType]; ok {
		return t, nil
	}
	return "", ErrUnknownType
}

func (me *renderer) Tag(d Domain) string {
	label := "`"

	label += `domain:"` + d.Name + `"`

	if len(d.Type) > 0 {
		label += `,type:"` + d.Type + `"`
	}

	label += "`"

	return label
}

// get struct name
func (me *renderer) StructName(parts ...string) string {
	return parts[0] + public(parts[1:]...)
}

func clean(body string) (res string) {
	return strings.Replace(body, "\r", "", -1)
}

// match private variable
func private(parts ...string) string {
	return export(regexp.MustCompile(`[-_]\w`), parts...)
}

// match public variable
func public(parts ...string) string {
	return export(regexp.MustCompile(`^\w|[-_]\w`), parts...)
}

func export(delim *regexp.Regexp, parts ...string) (res string) {
	for _, in := range parts {

		// replace letter
		res += delim.ReplaceAllStringFunc(in, func(match string) string {
			switch len(match) {
			// direct upper letter
			case 1:
				return strings.ToUpper(match)
			// filter -_ character, then upper letter
			case 2:
				return strings.ToUpper(match[1:])
			}
			panic("unreachable")
		})
	}

	return
}

func main() {
	var r renderer

	spec, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln("Please pass spec on stdin", err)
	}

	err = xml.Unmarshal(spec, &r.Root)

	if err != nil {
		log.Fatalln("Could not parse XML:", err)
	}

	if err = packageTemplate.Execute(os.Stdout, &r); err != nil {
		log.Fatalln("Generate error: ", err)
	}
}
