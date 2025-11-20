package fileops

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)
func WriteValueToFile(value float32 ,fileName string) {
	valueText := fmt.Sprint(value)
	os.WriteFile(fileName, []byte(valueText), 0644)
}

func GetFloatFromFile(fileName string) (float32, error) {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return 1000, errors.New("No stored balance found, starting with 1000")
	}

	valueText := string(data)

	value, err := strconv.ParseFloat(valueText, 64)
	if err != nil {
		return 1000, errors.New("Failed to parse stored value")
	}

	return float32(value), nil
}