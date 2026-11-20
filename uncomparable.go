// Copyright (c) 2026 SUDOLESS SRL
// SPDX-License-Identifier: BSD-3-Clause-Clear

package rapid

import (
	"fmt"
	"reflect"
)

// mayBeUncomparable reports whether a value of the comparable type typ can still fail to compare:
// an interface can hold a slice, map or func, and so can a struct or array that holds an interface.
func mayBeUncomparable(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Interface:
		return true
	case reflect.Array:
		return mayBeUncomparable(typ.Elem())
	case reflect.Struct:
		for field := range typ.Fields() {
			if mayBeUncomparable(field.Type) {
				return true
			}
		}
	}
	return false
}

// checkKey panics, naming gen and the dynamic type of key, when key cannot be compared,
// which a map would report only as a hash error inside the runtime.
func checkKey[K comparable](gen fmt.Stringer, key K) {
	if !reflect.ValueOf(&key).Elem().Comparable() {
		panic(fmt.Sprintf("%v: a key of type %T cannot be compared, so it cannot be told apart from others; draw comparable values", gen, key))
	}
}
