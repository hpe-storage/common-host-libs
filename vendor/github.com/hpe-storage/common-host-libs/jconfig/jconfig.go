/*
(c) Copyright 2017 Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

const (
	// Int64Type represents int64 type
	Int64Type = "int64"
	// StringType represents string type
	StringType = "string"
	// BoolType represents bool type
	BoolType = "bool"
)

// Config contains a map loaded from t a json file
type Config struct {
	config map[string]interface{}
}

//NewConfig loads the JSON in the file referred to in the path
func NewConfig(path string) (*Config, error) {
	c := &Config{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if file != nil {
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&c.config); err != nil {
			return nil, err
		}
	}
	return c, nil
}

//GetString returns the string value loaded from the JSON (backward compatibility)
func (c *Config) GetString(key string) (s string) {
	s, _ = c.GetStringWithError(key)
	return
}

//GetStringWithError returns the string value loaded from the JSON
func (c *Config) GetStringWithError(key string) (s string, err error) {
	if _, found := c.config[key]; found {
		switch value := c.config[key].(type) {
		case string:
			return value, nil
		default:
			return fmt.Sprintf("%v", c.config[key]), nil
		}
	}
	return s, fmt.Errorf("key:%v not found", key)
}

//GetStringSlice returns the string value loaded from the JSON (backward compatibility)
func (c *Config) GetStringSlice(key string) (strings []string) {
	strings, _ = c.GetStringSliceWithError(key)
	return
}

//GetMapSlice returns map of  strings and interface with error
func (c *Config) GetMapSlice(key string) (maps []map[string]interface{}, err error) {
	if _, found := c.config[key]; found {
		switch values := c.config[key].(type) {
		case []interface{}:
			for _, value := range values {
				v := reflect.ValueOf(value)
				if v.Kind() == reflect.Map {
					for _, key := range v.MapKeys() {
						val := v.MapIndex(key).Interface()
						maps = append(maps, map[string]interface{}{fmt.Sprintf("%v", key): val})
					}
				}
			}
			return maps, nil
		}
	}
	return nil, fmt.Errorf("key:%v not found", key)
}

//GetMap returns map of  string and interface with error
func (c *Config) GetMap(key string) (keyMap map[string]interface{}, err error) {
	keyMap = make(map[string]interface{})
	if _, found := c.config[key]; found {
		switch value := c.config[key].(type) {
		case interface{}:
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Map {
				for _, key := range v.MapKeys() {
					val := v.MapIndex(key).Interface()
					keyMap[fmt.Sprintf("%v", key)] = val
				}
				return keyMap, nil
			}
		}
	}
	return nil, fmt.Errorf("key:%v not found", key)
}

// GetValueFromMapByType returns value of given key in the map with given valueType
func GetValueFromMapByType(optionMap map[string]interface{}, key string, valueType string) (value interface{}, err error) {
	if value, ok := optionMap[key]; ok {
		var typeValue interface{}
		switch valueType {
		case Int64Type:
			typeValue, err = getInt64Type(value)
			if err != nil {
				return nil, err
			}
		case StringType:
			typeValue, err = getStringType(value)
			if err != nil {
				return nil, err
			}
		case BoolType:
			typeValue, err = getBoolType(value)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown option type %v provided to populate", valueType)

		}
		return typeValue, nil
	}
	return nil, fmt.Errorf("cannot find key %v in the option map", key)
}

// get int64 type value from given interface type
func getInt64Type(value interface{}) (int64, error) {
	switch value := value.(type) {
	//json marshall stores numbers as floats
	case float64:
		return int64(value), nil
	//we can always try to parse a string
	case string:
		return strconv.ParseInt(value, 10, 64)
	default:
		return 0, fmt.Errorf("value:%v is not a number kind:%s type:%s", value, reflect.TypeOf(value).Kind(), reflect.TypeOf(value))
	}
}

// get bool type value from given interface type
func getBoolType(value interface{}) (bool, error) {
	switch value := value.(type) {
	case bool:
		return value, nil
	//we can always try to parse a string
	case string:
		return strconv.ParseBool(value)
	default:
		return false, fmt.Errorf("value:%v is not a number kind:%s type:%s", value, reflect.TypeOf(value).Kind(), reflect.TypeOf(value))
	}
}

// get string type value from given interface type
func getStringType(value interface{}) (string, error) {
	switch value := value.(type) {
	case string:
		return value, nil
		//we can always try to parse a string
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

//GetStringSliceWithError returns the string value loaded from the JSON
func (c *Config) GetStringSliceWithError(key string) (strings []string, err error) {
	if _, found := c.config[key]; found {
		switch value := c.config[key].(type) {
		case []interface{}:
			for _, d := range value {
				strings = append(strings, fmt.Sprintf("%v", d))
			}
			return strings, nil
		default:
			return strings, fmt.Errorf("key:%v is not a slice.  value:%v kind:%s type:%s", key, c.config[key], reflect.TypeOf(c.config[key]).Kind(), reflect.TypeOf(c.config[key]))
		}
	}
	return strings, fmt.Errorf("key:%v not found", key)
}

//GetInt64 returns the value in the JSON cast to int64 (backward compatibility)
func (c *Config) GetInt64(key string) (i int64) {
	i, _ = c.GetInt64SliceWithError(key)
	return
}

//GetInt64SliceWithError returns the value in the JSON cast to int64
func (c *Config) GetInt64SliceWithError(key string) (i int64, err error) {
	if _, found := c.config[key]; found {
		switch value := c.config[key].(type) {
		//json marshall stores numbers as floats
		case float64:
			return int64(value), nil
		//we can always try to parse a string
		case string:
			return strconv.ParseInt(value, 10, 64)
		default:
			return 0, fmt.Errorf("key:%v is not a number.  value:%v kind:%s type:%s", key, c.config[key], reflect.TypeOf(c.config[key]).Kind(), reflect.TypeOf(c.config[key]))
		}
	}
	return 0, fmt.Errorf("key:%v not found", key)
}

//GetBool returns the value in the JSON cast to bool
func (c *Config) GetBool(key string) (b bool, err error) {
	if _, found := c.config[key]; found {
		switch value := c.config[key].(type) {
		case bool:
			return bool(value), nil
		//we can always try to parse a string
		case string:
			return strconv.ParseBool(value)
		default:
			return false, fmt.Errorf("key:%v is not a bool.  value:%v kind:%s type:%s", key, c.config[key], reflect.TypeOf(c.config[key]).Kind(), reflect.TypeOf(c.config[key]))
		}
	}
	return false, fmt.Errorf("key:%v not found", key)
}
