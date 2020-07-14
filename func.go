package frame

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

/**
切片变成字符串切片
 */
func convertSliceToStrings(s interface{}) []string {
	result := make([]string, 0)
	switch s.(type) {
	case []interface{}:
		for _, v := range s.([]interface{}) {
			result = append(result, convertToString(v))
		}
	case []int:
		for _, v := range s.([]int) {
			result = append(result, convertToString(v))
		}
	case []string:
		for _, v := range s.([]string) {
			result = append(result, convertToString(v))
		}
	case []float64:
		for _, v := range s.([]float64) {
			result = append(result, convertToString(v))
		}
	case []float32:
		for _, v := range s.([]float32) {
			result = append(result, convertToString(v))
		}
	case []int8:
		for _, v := range s.([]int8) {
			result = append(result, convertToString(v))
		}
	case []int16:
		for _, v := range s.([]int16) {
			result = append(result, convertToString(v))
		}
	case []int32:
		for _, v := range s.([]int32) {
			result = append(result, convertToString(v))
		}
	case []int64:
		for _, v := range s.([]int64) {
			result = append(result, convertToString(v))
		}
	case []bool:
		for _, v := range s.([]bool) {
			result = append(result, convertToString(v))
		}
	default:
		return result
	}
	return result
}

/**
将切片分成多个切片
*/
func arrayChunk(data []map[string]interface{}, onceMaxCount int) [][]map[string]interface{} {
	beginIndex := 0
	endIndex := 0
	maxLen := len(data)
	result := make([][]map[string]interface{}, 0)
	for {
		if beginIndex < maxLen {
			endIndex = beginIndex + onceMaxCount
			if endIndex > maxLen {
				endIndex = maxLen
			}
			buffer := make([]map[string]interface{}, 0)
			for i := beginIndex; i < endIndex; i++ {
				buffer = append(buffer, data[i])
			}
			beginIndex = endIndex
			result = append(result, buffer)
		} else {
			break
		}
	}
	return result
}

/**
字符串加转义
 */
func addSlashes(str string) string {
	tmpRune := make([]rune, 0)
	strRune := []rune(str)
	for _, ch := range strRune {
		switch ch {
		case []rune{'\\'}[0], []rune{'"'}[0], []rune{'\''}[0]:
			tmpRune = append(tmpRune, []rune{'\\'}[0])
			tmpRune = append(tmpRune, ch)
		default:
			tmpRune = append(tmpRune, ch)
		}
	}
	return string(tmpRune)
}

/**
任意字符变成字符串
 */
func convertToString(v interface{}) string {
	switch v.(type) {
	case int:
		return strconv.Itoa(v.(int))
	case string:
		return v.(string)
	case int8:
		return strconv.Itoa(int(v.(int8)))
	case int16:
		return strconv.Itoa(int(v.(int16)))
	case int32:
		return strconv.Itoa(int(v.(int32)))
	case int64:
		return strconv.Itoa(int(v.(int64)))
	case float32:
		return fmt.Sprintf("%.2f", v.(float32))
	case float64:
		return fmt.Sprintf("%.2f", v.(float64))
	case bool:
		if v.(bool) {
			return "true"
		} else {
			return "false"
		}
	default:
		return ""
	}
}

//json非转义
func jsonMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

// 判断文件夹是否存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}