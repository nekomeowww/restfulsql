package restfulsql

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	query := `["AND", ["a", ["AND", ["b", ["OR", ["c"], [2]]], [2]]], [10, ""]]`
	parser := NewRestfulSQLParser(query)
	rsql, err := parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	assert.Equal("AND", rsql.Mode)
	require.NotEmpty(rsql.Fields)
	require.Len(rsql.Fields, 2)
	assert.Equal([]interface{}{
		"AND",
		[]interface{}{"b", []interface{}{"OR", []interface{}{"c"}, []interface{}{2.0}}},
		[]interface{}{2.0},
	}, rsql.Fields[1].([]interface{}))
	assert.Equal([]interface{}{10.0, ""}, rsql.Values)
}

func TestCompile(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	query := `["AND", ["a", "b"], [2, "2"]]`
	parser := NewRestfulSQLParser(query)
	rsql, err := parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	compiledStr, err := parser.Compile()
	require.NoError(err)
	require.NotEmpty(compiledStr)
	assert.Equal("(a = 2) AND (b = '2')", compiledStr)

	query = `["AND", ["b", ["OR", ["c"], [2]]], [2, ""]]`
	parser = NewRestfulSQLParser(query)
	rsql, err = parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	compiledStr, err = parser.Compile()
	require.NoError(err)
	require.NotEmpty(compiledStr)
	assert.Equal("(b = 2) AND ((c = 2))", compiledStr)

	query = `["AND", ["a", ["AND", ["b", ["OR", ["c", "d"], [2, 3]]], [2, ""]]], [10, ""]]`
	parser = NewRestfulSQLParser(query)
	rsql, err = parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	compiledStr, err = parser.Compile()
	require.NoError(err)
	require.NotEmpty(compiledStr)
	assert.Equal("(a = 10) AND ((b = 2) AND ((c = 2) OR (d = 3)))", compiledStr)
}

func TestNestedSQLCount(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	query := `["AND", ["a", ["AND", ["b", ["OR", ["c"], [2]]], [2, ""]]], [10, ""]]`
	parser := NewRestfulSQLParser(query)
	rsql, err := parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	fields := rsql.Fields[1]
	count, index := findNestedRSQL(fields)
	assert.Equal(1, count)
	assert.Len(index, 1)
	fields = rsql.Fields[1].([]interface{})[1]
	count, index = findNestedRSQL(fields)
	assert.Equal(1, count)
	assert.Len(index, 1)
	fields = rsql.Fields[1].([]interface{})[1].([]interface{})[1]
	count, index = findNestedRSQL(fields)
	assert.Equal(0, count)
	assert.Len(index, 0)
}

func TestContainsType(t *testing.T) {
	assert := assert.New(t)

	assert.True(containsType([]interface{}{
		"a",
		[]interface{}{"b", []interface{}{"c"}},
	}, reflect.TypeOf([]interface{}{})))
}

func TestBuildString(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	query := `["AND", ["a"], [2]]`
	parser := NewRestfulSQLParser(query)
	rsql, err := parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	str := buildString(rsql.Mode, rsql.Fields, rsql.Values)
	assert.Equal("(a = 2)", str)

	query = `["AND", ["a", "b"], [2, "2"]]`
	parser = NewRestfulSQLParser(query)
	rsql, err = parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	str = buildString(rsql.Mode, rsql.Fields, rsql.Values)
	assert.Equal("(a = 2) AND (b = '2')", str)
}

func TestFalttenNestedRSQL(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	query := `["AND", ["a", "b"], [2, "2"]]`
	parser := NewRestfulSQLParser(query)
	rsql, err := parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	count, index := findNestedRSQL(rsql.Fields)
	assert.Zero(count)
	assert.Empty(index)
	err = falttenNestedRSQL(rsql.Fields, index)
	require.NoError(err)

	query = `["AND", ["b", ["OR", ["c"], [2]]], [2, ""]]`
	parser = NewRestfulSQLParser(query)
	rsql, err = parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	count, index = findNestedRSQL(rsql.Fields)
	assert.Equal(1, count)
	assert.Len(index, 1)
	assert.ElementsMatch([]int{1}, index)
	err = falttenNestedRSQL(rsql.Fields, index)
	require.NoError(err)
	assert.Equal([]interface{}{"b", "(c = 2)"}, rsql.Fields)

	query = `["AND", ["a", ["AND", ["b", ["OR", ["c", "d"], [2, 3]]], [2, ""]]], [10, ""]]`
	parser = NewRestfulSQLParser(query)
	rsql, err = parser.Parse()
	require.NoError(err)
	require.NotNil(rsql)

	count, index = findNestedRSQL(rsql.Fields)
	assert.Equal(1, count)
	assert.Len(index, 1)
	assert.ElementsMatch([]int{1}, index)

	err = falttenNestedRSQL(rsql.Fields, index)
	require.NoError(err)
	assert.Equal([]interface{}{"a", "(b = 2) AND ((c = 2) OR (d = 3))"}, rsql.Fields)
}
