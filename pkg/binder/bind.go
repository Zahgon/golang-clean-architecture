// Package binder binds request data into structs for gin handlers.
//
// Binding follows the rules the API has always used: path params (`param` tag), query params (`query` tag, only for
// GET/DELETE/HEAD) and then the request body chosen by Content-Type. Errors are returned as *HTTPError with the same
// messages clients received before, gin's own binding is not used because its rules and messages differ.
package binder

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	mimeApplicationJSON = "application/json"
	mimeApplicationXML  = "application/xml"
	mimeTextXML         = "text/xml"
	mimeApplicationForm = "application/x-www-form-urlencoded"
	mimeMultipartForm   = "multipart/form-data"

	defaultMemory = 32 << 20 // 32 MB
)

// BindUnmarshaler is the interface used to wrap the UnmarshalParam method.
// Types that don't implement this, but do implement encoding.TextUnmarshaler
// will use that interface instead.
type BindUnmarshaler interface {
	// UnmarshalParam decodes and assigns a value from an form or query param.
	UnmarshalParam(param string) error
}

// bindMultipleUnmarshaler is used by binder to unmarshal multiple values from request at once to
// type implementing this interface. For example request could have multiple query fields `?a=1&a=2&b=test` in that case
// for `a` following slice `["1", "2"] will be passed to unmarshaller.
type bindMultipleUnmarshaler interface {
	UnmarshalParams(params []string) error
}

// Bind binds path params, query params (GET/DELETE/HEAD only) and the request body, in that order, into i.
// Each step could override previous step bound values.
func Bind(c *gin.Context, i any) (err error) {
	if err := bindPathParams(c, i); err != nil {
		return err
	}
	// Only bind query parameters for GET/DELETE/HEAD to avoid unexpected behavior with destination struct binding from body.
	// For example a request URL `&id=1&lang=en` with body `{"id":100,"lang":"de"}` would lead to precedence issues.
	method := c.Request.Method
	if method == http.MethodGet || method == http.MethodDelete || method == http.MethodHead {
		if err = bindQueryParams(c, i); err != nil {
			return err
		}
	}
	return bindBody(c, i)
}

// bindPathParams binds path params to bindable object
func bindPathParams(c *gin.Context, i any) error {
	params := map[string][]string{}
	for _, param := range c.Params {
		params[param.Key] = []string{param.Value}
	}
	if err := bindData(i, params, "param"); err != nil {
		return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	return nil
}

// bindQueryParams binds query params to bindable object
func bindQueryParams(c *gin.Context, i any) error {
	if err := bindData(i, c.Request.URL.Query(), "query"); err != nil {
		return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	return nil
}

// bindBody binds request body contents to bindable object
// NB: then binding forms take note that this implementation uses standard library form parsing
// which parses form data from BOTH URL and BODY if content type is not MIMEMultipartForm
// See non-MIMEMultipartForm: https://golang.org/pkg/net/http/#Request.ParseForm
// See MIMEMultipartForm: https://golang.org/pkg/net/http/#Request.ParseMultipartForm
func bindBody(c *gin.Context, i any) (err error) {
	req := c.Request
	if req.ContentLength == 0 {
		return
	}

	ctype := req.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ctype, mimeApplicationJSON):
		err = json.NewDecoder(req.Body).Decode(i)
		if ute, ok := err.(*json.UnmarshalTypeError); ok {
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
		} else if se, ok := err.(*json.SyntaxError); ok {
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
		} else if err != nil {
			return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	case strings.HasPrefix(ctype, mimeApplicationXML), strings.HasPrefix(ctype, mimeTextXML):
		if err = xml.NewDecoder(req.Body).Decode(i); err != nil {
			if ute, ok := err.(*xml.UnsupportedTypeError); ok {
				return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unsupported type error: type=%v, error=%v", ute.Type, ute.Error())).SetInternal(err)
			} else if se, ok := err.(*xml.SyntaxError); ok {
				return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: line=%v, error=%v", se.Line, se.Error())).SetInternal(err)
			}
			return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	case strings.HasPrefix(ctype, mimeApplicationForm), strings.HasPrefix(ctype, mimeMultipartForm):
		params, err := formParams(req)
		if err != nil {
			return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		if err = bindData(i, params, "form"); err != nil {
			return NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	default:
		return ErrUnsupportedMediaType
	}
	return nil
}

func formParams(req *http.Request) (url.Values, error) {
	if strings.HasPrefix(req.Header.Get("Content-Type"), mimeMultipartForm) {
		if err := req.ParseMultipartForm(defaultMemory); err != nil {
			return nil, err
		}
	} else {
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
	}
	return req.Form, nil
}

// bindData will bind data ONLY fields in destination struct that have EXPLICIT tag
func bindData(destination any, data map[string][]string, tag string) error {
	if destination == nil || len(data) == 0 {
		return nil
	}
	typ := reflect.TypeOf(destination).Elem()
	val := reflect.ValueOf(destination).Elem()

	// Support binding to limited Map destinations:
	// - map[string][]string,
	// - map[string]string <-- (binds first value from data slice)
	// - map[string]interface{}
	// You are better off binding to struct but there are user who want this map feature. Source of data for these cases are:
	// params,query,header,form as these sources produce string values, most of the time slice of strings, actually.
	if typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String {
		k := typ.Elem().Kind()
		isElemInterface := k == reflect.Interface
		isElemString := k == reflect.String
		isElemSliceOfStrings := k == reflect.Slice && typ.Elem().Elem().Kind() == reflect.String
		if !(isElemSliceOfStrings || isElemString || isElemInterface) {
			return nil
		}
		if val.IsNil() {
			val.Set(reflect.MakeMap(typ))
		}
		for k, v := range data {
			if isElemString {
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
			} else {
				val.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}
		return nil
	}

	// !struct
	if typ.Kind() != reflect.Struct {
		if tag == "param" || tag == "query" || tag == "header" {
			// incompatible type, data is probably to be found in the body
			return nil
		}
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if typeField.Anonymous {
			if structField.Kind() == reflect.Ptr {
				structField = structField.Elem()
			}
		}
		if !structField.CanSet() {
			continue
		}
		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)
		if typeField.Anonymous && structFieldKind == reflect.Struct && inputFieldName != "" {
			// if anonymous struct with query/param/form tags, report an error
			return errors.New("query/param/form tags are not allowed with anonymous struct field")
		}

		if inputFieldName == "" {
			// If tag is nil, we inspect if the field is a not BindUnmarshaler struct and try to bind data into it (might contains fields with tags).
			// structs that implement BindUnmarshaler are bound only when they have explicit tag
			if _, ok := structField.Addr().Interface().(BindUnmarshaler); !ok && structFieldKind == reflect.Struct {
				if err := bindData(structField.Addr().Interface(), data, tag); err != nil {
					return err
				}
			}
			// does not have explicit tag and is not an ordinary struct - so move to next field
			continue
		}

		inputValue, exists := data[inputFieldName]
		if !exists {
			// Go json.Unmarshal supports case insensitive binding.  However the
			// url params are bound case sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			for k, v := range data {
				if strings.EqualFold(k, inputFieldName) {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			continue
		}

		// NOTE: algorithm here is not particularly sophisticated. It probably does not work with absurd types like `**[]*int`
		// but it is smart enough to handle niche cases like `*int`,`*[]string`,`[]*int` .

		// try unmarshalling first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalInputsToField(typeField.Type.Kind(), inputValue, structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		if ok, err := unmarshalInputToField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		// we could be dealing with pointer to slice `*[]string` so dereference it. There are wierd OpenAPI generators
		// that could create struct fields like that.
		if structFieldKind == reflect.Pointer {
			structFieldKind = structField.Elem().Kind()
			structField = structField.Elem()
		}

		if structFieldKind == reflect.Slice {
			sliceOf := structField.Type().Elem().Kind()
			numElems := len(inputValue)
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return err
				}
			}
			structField.Set(slice)
			continue
		}

		if err := setWithProperType(structFieldKind, inputValue[0], structField); err != nil {
			return err
		}
	}
	return nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	// But also call it here, in case we're dealing with an array of BindUnmarshalers
	if ok, err := unmarshalInputToField(valueKind, val, structField); ok {
		return err
	}

	switch valueKind {
	case reflect.Ptr:
		return setWithProperType(structField.Elem().Kind(), val, structField.Elem())
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	default:
		return errors.New("unknown type")
	}
	return nil
}

func unmarshalInputsToField(valueKind reflect.Kind, values []string, field reflect.Value) (bool, error) {
	if valueKind == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	fieldIValue := field.Addr().Interface()
	unmarshaler, ok := fieldIValue.(bindMultipleUnmarshaler)
	if !ok {
		return false, nil
	}
	return true, unmarshaler.UnmarshalParams(values)
}

func unmarshalInputToField(valueKind reflect.Kind, val string, field reflect.Value) (bool, error) {
	if valueKind == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	fieldIValue := field.Addr().Interface()
	switch unmarshaler := fieldIValue.(type) {
	case BindUnmarshaler:
		return true, unmarshaler.UnmarshalParam(val)
	case encoding.TextUnmarshaler:
		return true, unmarshaler.UnmarshalText([]byte(val))
	}

	return false, nil
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}
