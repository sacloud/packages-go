// Copyright 2022 The sacloud/packages-go Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validate

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-multierror"
)

var validatorInstance *validator.Validate

func init() {
	validatorInstance = validator.New()
	validatorInstance.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("name"), ",", 2)[0]
		if name == "" {
			// nameタグがない場合はyamlタグを参照
			name = strings.SplitN(fld.Tag.Get("yaml"), ",", 2)[0]
		}
		if name == "-" {
			return ""
		}
		return name
	})
}

func validate(v interface{}) error {
	return validatorInstance.Struct(v)
}

// RegisterCollectionValidator 要素/コレクションを示す単数名/複数名に対しそれぞれoneof/diveエイリアスを登録する
//
// ("zone", "zones", []string{"v1", "v2"}) とした場合、以下のエイリアスがバリデーターに登録される
//   - "zone"  => "oneof=v1 v2"のエイリアス
//   - "zones" => "dive,zone"のエイリアス
func RegisterCollectionValidator(singularName, pluralName string, allowedValues []string) {
	validatorInstance.RegisterAlias(singularName, fmt.Sprintf("oneof=%s", strings.Join(allowedValues, " ")))
	validatorInstance.RegisterAlias(pluralName, fmt.Sprintf("dive,%s", singularName))
}

// Struct 対象structを検証しerrorを返す
func Struct(v interface{}) error {
	errors := StructWithMultiError(v)
	if errors != nil {
		return errors.ErrorOrNil()
	}
	return nil
}

// StructWithMultiError 対象structを検証し、*multierror.Errorを返す
func StructWithMultiError(v interface{}) *multierror.Error {
	err := validate(v)
	if err != nil {
		if err != nil {
			// see https://github.com/go-playground/validator/blob/f6584a41c8acc5dfc0b62f7962811f5231c11530/_examples/simple/main.go#L59-L65
			if _, ok := err.(*validator.InvalidValidationError); ok {
				return &multierror.Error{Errors: []error{err}}
			}

			errors := &multierror.Error{}
			for _, err := range err.(validator.ValidationErrors) {
				errors = multierror.Append(errors, errorFromValidationErr(v, err))
			}
			return errors
		}
	}

	return nil
}

func errorFromValidationErr(target interface{}, err validator.FieldError) error {
	namespaces := strings.Split(err.Namespace(), ".")
	actualName := namespaces[len(namespaces)-1] // .で区切った末尾の要素

	param := err.Param()
	detail := err.ActualTag()
	if param != "" {
		detail += "=" + param
	}

	// detailがvalidatorのタグ名だけの場合の対応をここで行う。
	switch detail {
	case "file":
		detail = fmt.Sprintf("invalid file path: %v", err.Value())
	}

	return newError(actualName, detail)
}

func newError(name, message string) error {
	return fmt.Errorf("%s: %s", name, message)
}
