package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/piyushsingariya/syndicate/jsonschema"
	"github.com/piyushsingariya/syndicate/logger"
	"github.com/piyushsingariya/syndicate/models"
	"github.com/piyushsingariya/syndicate/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// IsValidSubcommand checks if the passed subcommand is supported by the parent command
func IsValidSubcommand(available []*cobra.Command, sub string) bool {
	for _, s := range available {
		if sub == s.CalledAs() {
			return true
		}
	}
	return false
}

func GetStreamsFromConfiguredCatalog(catalog *models.Catalog) []*models.Stream {
	result := []*models.Stream{}
	for _, stream := range catalog.Streams {
		result = append(result, stream.Stream)
	}

	return result
}

func ContainsValue(array interface{}, value interface{}) bool {
	arrValue := reflect.ValueOf(array)
	if arrValue.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < arrValue.Len(); i++ {
		item := arrValue.Index(i).Interface()
		if reflect.DeepEqual(item, value) {
			return true
		}
	}

	return false
}

func ToJSONSchema(obj interface{}) (string, error) {
	schema, err := jsonschema.Reflect(obj)
	if err != nil {
		return "", err
	}

	j, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		return "", err
	}

	return string(j), nil
}

func ToYamlSchema(obj interface{}) (string, error) {
	jsonSchema, err := ToJSONSchema(obj)
	if err != nil {
		return "", err
	}

	yamlData, err := yaml.JSONToYAML([]byte(jsonSchema))
	if err != nil {
		return "", err
	}

	return string(yamlData), nil
}

// Unmarshal serializes and deserializes any from into the object
// return error if occurred
func Unmarshal(from interface{}, object interface{}) error {
	reformatted := reformatInnerMaps(from)
	b, err := json.Marshal(reformatted)
	if err != nil {
		return fmt.Errorf("error marshalling object: %v", err)
	}
	err = json.Unmarshal(b, object)
	if err != nil {
		return fmt.Errorf("error unmarshalling from object: %v", err)
	}

	return nil
}

// reformatInnerMaps converts all map[interface{}]interface{} into map[string]interface{}
// because json.Marshal doesn't support map[interface{}]interface{} (supports only string keys)
// but viper produces map[interface{}]interface{} for inner maps
// return recursively converted all map[interface]interface{} to map[string]interface{}
func reformatInnerMaps(valueI interface{}) interface{} {
	switch value := valueI.(type) {
	case []interface{}:
		for i, subValue := range value {
			value[i] = reformatInnerMaps(subValue)
		}
		return value
	case map[interface{}]interface{}:
		newMap := make(map[string]interface{}, len(value))
		for k, subValue := range value {
			newMap[fmt.Sprint(k)] = reformatInnerMaps(subValue)
		}
		return newMap
	case map[string]interface{}:
		for k, subValue := range value {
			value[k] = reformatInnerMaps(subValue)
		}
		return value
	default:
		return valueI
	}
}

func CheckIfFilesExists(files ...string) error {
	for _, file := range files {
		// Check if the file or directory exists
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist", file)
		}

		_, err = os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s", file)
		}
	}

	return nil
}

func ReadFile(file string) interface{} {
	content, err := ReadFileE(file)
	if err != nil {
		logger.Error(err)
		return nil
	}

	return content
}

func ReadFileE(file string) (interface{}, error) {
	if err := CheckIfFilesExists(file); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("file not found : %s", err)
	}

	var content interface{}

	err = yaml.Unmarshal(data, &content)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func IsOfType(object interface{}, decidingKey string) (bool, error) {
	objectMap := make(map[string]interface{})
	if err := Unmarshal(object, &objectMap); err != nil {
		return false, err
	}

	if _, found := objectMap[decidingKey]; found {
		return true, nil
	}

	return false, nil
}

func StreamIdentifier(namespace, name string) string {
	return namespace + name
}

func AppendDataType(property *models.Property, typ types.DataType) {
	for _, tp := range property.Type {
		if tp == typ {
			return
		}
	}

	property.Type = append(property.Type, typ)
}

func ProcessDataTypes(stream *models.Stream, record models.Record) {
	/* TODO: check for datatype from values received from record
	Most of the times data will be processed here, determined what is the type of key and all
	Add into properties
	Determine primitive type and all
	*/
	for key := range stream.JSONSchema.Properties {
		if value, found := record.Data[key]; !found || value == nil {
			AppendDataType(stream.JSONSchema.Properties[key], types.Null)
		}
	}
}
