// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// use a single instance of Validate, it caches struct info
var validate *validator.Validate

// Validate validates the config once it has been loaded using runtimeConfig
func (c *Config) Validate() error {
	// Parse viper values into a runtimeConfig struct
	config := runtimeConfig{}
	typeOf := reflect.TypeOf(config)
	valueOf := reflect.ValueOf(&config).Elem()
	for i := 0; i < typeOf.NumField(); i++ {
		field := typeOf.Field(i)
		viperKey, ok := field.Tag.Lookup("viper")
		if !ok {
			panic("Unexpected missing viper tag on Config struct")
		}
		fieldType := valueOf.Field(i).Type()
		switch fieldType.Kind() {
		case reflect.Bool:
			valueOf.Field(i).SetBool(viper.GetBool(viperKey))
		case reflect.String:
			valueOf.Field(i).SetString(viper.GetString(viperKey))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			valueOf.Field(i).SetInt(viper.GetInt64(viperKey))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			valueOf.Field(i).SetUint(viper.GetUint64(viperKey))
		case reflect.Float32, reflect.Float64:
			valueOf.Field(i).SetFloat(viper.GetFloat64(viperKey))
		default:
			valueOf.Field(i).Set(reflect.ValueOf(viper.Get(viperKey)))
		}
	}
	// Use validator to validate it
	validate = validator.New(validator.WithRequiredStructEnabled())
	err := validate.Struct(config)
	if err != nil {
		msg := ""
		for _, err := range err.(validator.ValidationErrors) {
			msg += fmt.Sprintf("\n%s failed validation on '%s' validator.", err.Field(), err.Tag())
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}
