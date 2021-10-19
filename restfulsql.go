package restfulsql

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// RSQL 基础结构
type RSQL struct {
	Mode   string
	Fields []interface{}
	Values []interface{}

	refFields       reflect.Value
	refValues       reflect.Value
	refFieldsLength int
	refValuesLength int

	nestedRSQLCount   int
	nestedRSQLIndexes []int
}

// Parser 解析器
type Parser struct {
	Query *RSQL

	rawQuery string
}

// 错误
var (
	ErrInvalidRestfulSQL          = errors.New("invalid restful sql")
	ErrNumOfFieldAndValueMismatch = errors.New("number of fields and values mismatched")
)

// NewRestfulSQLParser 新建一个 RestfulSQLParser 解析器
func NewRestfulSQLParser(query string) *Parser {
	return &Parser{
		rawQuery: query,
	}
}

// Parse 解析 SQL
func (r *Parser) Parse() (*RSQL, error) {
	err := json.Unmarshal([]byte(r.rawQuery), &r.Query)
	if err != nil {
		return nil, err
	}

	err = r.Query.checkLength()
	if err != nil {
		return nil, err
	}

	return r.Query, nil
}

func (r *Parser) Compile() (string, error) {
	err := falttenNestedRSQL(r.Query.Fields, r.Query.nestedRSQLIndexes)
	if err != nil {
		return "", err
	}

	return buildString(r.Query.Mode, r.Query.Fields, r.Query.Values), nil
}

func (r *RSQL) checkLength() error {
	if r.refFieldsLength != r.refValuesLength {
		return ErrNumOfFieldAndValueMismatch
	}

	return nil
}

// TODO: 字段名称检查
// func (r *RSQL) checkFieldsDuplication() error {
// 	return nil
// }

// UnmarshalJSON 反序列化
func (r *RSQL) UnmarshalJSON(data []byte) error {
	var raw []interface{}
	err := json.Unmarshal([]byte(data), &raw)
	if err != nil {
		return err
	}

	parsedRSQL, err := unmarshalToRSQLStruct(raw)
	if err != nil {
		return err
	}

	// TODO: 优化赋值
	r.Fields = parsedRSQL.Fields
	r.Values = parsedRSQL.Values
	r.Mode = parsedRSQL.Mode
	r.refFields = parsedRSQL.refFields
	r.refValues = parsedRSQL.refValues
	r.refFieldsLength = parsedRSQL.refFieldsLength
	r.refValuesLength = parsedRSQL.refValuesLength
	r.nestedRSQLCount = parsedRSQL.nestedRSQLCount
	r.nestedRSQLIndexes = parsedRSQL.nestedRSQLIndexes
	return nil
}

func unmarshalToRSQLStruct(raw []interface{}) (*RSQL, error) {
	var r RSQL
	if len(raw) != 3 {
		return nil, ErrInvalidRestfulSQL
	}

	var ok bool
	r.Mode, ok = raw[0].(string)
	if !ok {
		return nil, ErrInvalidRestfulSQL
	}

	r.Fields, ok = raw[1].([]interface{})
	if !ok {
		return nil, ErrInvalidRestfulSQL
	}

	r.Values, ok = raw[2].([]interface{})
	if !ok {
		return nil, ErrInvalidRestfulSQL
	}

	r.refFields = reflect.ValueOf(r.Fields)
	r.refValues = reflect.ValueOf(r.Values)
	r.refFieldsLength = r.refFields.Len()
	r.refValuesLength = r.refValues.Len()
	r.nestedRSQLCount, r.nestedRSQLIndexes = findNestedRSQL(r.Fields)
	return &r, nil
}

func findNestedRSQL(fields interface{}) (int, []int) {
	refFields := reflect.ValueOf(fields)
	nestedFieldCount := 0
	nestedFieldIndexes := make([]int, 0)
	for i := 0; i < refFields.Len(); i++ {
		v, ok := refFields.Index(i).Interface().([]interface{})
		if ok && containsType(v, reflect.TypeOf([]interface{}{})) {
			nestedFieldIndexes = append(nestedFieldIndexes, i)
			nestedFieldCount++
		}
	}

	return nestedFieldCount, nestedFieldIndexes
}

func containsType(fields interface{}, t reflect.Type) bool {
	refFields := reflect.ValueOf(fields)
	for i := 0; i < refFields.Len(); i++ {
		v := refFields.Index(i)
		if fmt.Sprintf("%T", v.Interface()) == t.String() {
			return true
		}
	}
	return false
}

func buildString(mode string, fields, values interface{}) string {
	fieldsSlice := fields.([]interface{})
	valuesSlice := values.([]interface{})
	queries := make([]string, len(fieldsSlice))
	for i, v := range fieldsSlice {
		if fmt.Sprintf("%T", valuesSlice[i]) == "string" {
			valueStr := valuesSlice[i].(string)
			if valueStr == "" {
				queries[i] = fmt.Sprintf("(%s)", v)
			} else {
				// TODO: 如果 fields len = 1 的话这个地方不应该再去弄一个括号
				queries[i] = fmt.Sprintf("(%v = '%v')", v, valueStr)
			}
		} else {
			// TODO: 如果 fields len = 1 的话这个地方不应该再去弄一个括号
			queries[i] = fmt.Sprintf("(%v = %v)", v, valuesSlice[i])
		}
	}

	return strings.Join(queries, fmt.Sprintf(" %s ", mode))
}

func falttenNestedRSQL(fields interface{}, nestedFieldsIndexes []int) error {
	fieldsSlice := fields.([]interface{})
	for _, v := range nestedFieldsIndexes {
		targetingFields := fieldsSlice[1].([]interface{})
		count, indexes := findNestedRSQL(targetingFields[v])
		if count > 0 {
			err := falttenNestedRSQL(targetingFields[v], indexes)
			if err != nil {
				return err
			}
		}

		rsql, err := unmarshalToRSQLStruct(targetingFields)
		if err != nil {
			return err
		}
		err = rsql.checkLength()
		if err != nil {
			return err
		}

		str := buildString(rsql.Mode, rsql.Fields, rsql.Values)
		fieldsSlice[v] = str
	}

	return nil
}
