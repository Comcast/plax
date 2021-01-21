/*
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
package dsl

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Comcast/plax/dsl"
)

const (
	// PluginDefNameKey of the PluginDef map
	PluginDefNameKey = "Name"
	// PluginDefDirKey of the PluginDef map
	PluginDefDirKey = "Dir"
	// PluginDefFilenameKey of the PluginDef map
	PluginDefFilenameKey = "Filename"
	// PluginDefParamsKey of the PluginDef map
	PluginDefParamsKey = "Params"
	// PluginDefSeedKey of the PluginDef map
	PluginDefSeedKey = "Seed"
	// PluginDefVerboseKey of the PluginDef map
	PluginDefVerboseKey = "Verbose"
	// PluginDefPriorityKey of the PluginDef map
	PluginDefPriorityKey = "Priority"
	// PluginDefLabelsKey of the PluginDef map
	PluginDefLabelsKey = "Labels"
	// PluginDefLogLevelKey of the PluginDef map
	PluginDefLogLevelKey = "LogLevels"
	// PluginDefListKey of the PluginDef map
	PluginDefListKey = "List"
	// PluginDefEmitJSONKey of the PluginDef map
	PluginDefEmitJSONKey = "EmitJSON"
	// PluginDefNonzeroOnAnyErrorKey of the PluginDef map
	PluginDefNonzeroOnAnyErrorKey = "NonzeroOnAnyErrorKey"
	// PluginDefRetryKey of the PluginDef map
	PluginDefRetryKey = "Retry"
)

var (
	// DefaultPluginModule to load; "There can be only one!"
	DefaultPluginModule PluginModule = "github.com/Comcast/plax"
)

// GetPluginDefName returns the Name
func (pd PluginDef) GetPluginDefName() (string, error) {
	value, ok := pd[PluginDefNameKey]
	if !ok {
		return "", fmt.Errorf("%s was not provided in the plugin definition", PluginDefNameKey)
	}

	ret, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", PluginDefNameKey)
	}

	return ret, nil
}

// GetPluginDefDir returns the Dir
func (pd PluginDef) GetPluginDefDir() (string, error) {
	value, ok := pd[PluginDefDirKey]
	if !ok {
		return "", fmt.Errorf("%s was not provided in the plugin definition", PluginDefDirKey)
	}

	ret, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", PluginDefDirKey)
	}

	return ret, nil
}

// GetPluginDefFilename returns the Filename
func (pd PluginDef) GetPluginDefFilename() (string, error) {
	value, ok := pd[PluginDefFilenameKey]
	if !ok {
		return "", fmt.Errorf("%s was not provided in the plugin definition", PluginDefFilenameKey)
	}

	ret, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", PluginDefFilenameKey)
	}

	return ret, nil
}

// GetPluginDefParams returns the Params
func (pd PluginDef) GetPluginDefParams() (map[string]interface{}, error) {
	value, ok := pd[PluginDefParamsKey]
	if !ok {
		return nil, fmt.Errorf("%s was not provided in the plugin definition", PluginDefParamsKey)
	}

	bss, ok := value.(*dsl.Bindings)
	if !ok {
		return nil, fmt.Errorf("%s is not a map[string]interface{}", PluginDefParamsKey)
	}

	ret := make(map[string]interface{})
	for k, v := range *bss {
		ret[k] = v
	}

	return ret, nil
}

// GetPluginDefSeed returns the Seed
func (pd PluginDef) GetPluginDefSeed() (int64, error) {
	value, ok := pd[PluginDefSeedKey]
	if !ok {
		return -1, fmt.Errorf("%s was not provided in the plugin definition", PluginDefSeedKey)
	}

	ret, ok := value.(int64)
	if !ok {
		return -1, fmt.Errorf("%s is not a int64", PluginDefSeedKey)
	}

	return ret, nil
}

// GetPluginDefVerbose returns the Filename
func (pd PluginDef) GetPluginDefVerbose() (bool, error) {
	value, ok := pd[PluginDefVerboseKey]
	if !ok || value == nil {
		return false, fmt.Errorf("%s was not provided in the plugin definition", PluginDefVerboseKey)
	}

	ret, ok := value.(*bool)
	if !ok {
		return false, fmt.Errorf("%s is not a bool", PluginDefVerboseKey)
	}

	return *ret, nil
}

// GetPluginDefLogLevel returns the Filename
func (pd PluginDef) GetPluginDefLogLevel() (string, error) {
	value, ok := pd[PluginDefLogLevelKey]
	if !ok || value == nil {
		return "", fmt.Errorf("%s was not provided in the plugin definition", PluginDefLogLevelKey)
	}

	ret, ok := value.(*string)
	if !ok {
		return "", fmt.Errorf("%s is not a string", PluginDefLogLevelKey)
	}

	return *ret, nil
}

// GetPluginDefPriorty returns the Priority
func (pd PluginDef) GetPluginDefPriorty() (int, error) {
	value, ok := pd[PluginDefPriorityKey]
	if !ok {
		return -1, fmt.Errorf("%s was not provided in the plugin definition", PluginDefPriorityKey)
	}

	ret, ok := value.(int)
	if !ok {
		return -1, fmt.Errorf("%s is not a int", PluginDefPriorityKey)
	}

	return ret, nil
}

// GetPluginDefLabels returns the Labels
func (pd PluginDef) GetPluginDefLabels() (string, error) {
	value, ok := pd[PluginDefLabelsKey]
	if !ok {
		return "", fmt.Errorf("%s was not provided in the plugin definition", PluginDefLabelsKey)
	}

	ret, ok := value.([]string)
	if !ok {
		return "", fmt.Errorf("%s is not a []string", PluginDefLabelsKey)
	}

	return strings.Join(ret, ","), nil
}

// GetPluginDefRetry returns the Retry
func (pd PluginDef) GetPluginDefRetry() (string, error) {
	value, ok := pd[PluginDefRetryKey]
	if !ok {
		return "", fmt.Errorf("%s was not provided in the plugin definition", PluginDefRetryKey)
	}

	ret, ok := value.(int)
	if !ok {
		return "", fmt.Errorf("%s is not a int", PluginDefRetryKey)
	}

	return strconv.Itoa(ret), nil
}

// GetPluginDefList returns the List flag
func (pd PluginDef) GetPluginDefList() (bool, error) {
	value, ok := pd[PluginDefListKey]
	if !ok {
		return false, nil
	}

	ret, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s is not a bool", PluginDefListKey)
	}

	return ret, nil
}

// GetPluginDefEmitJSON returns the EmitJSON flag
func (pd PluginDef) GetPluginDefEmitJSON() (bool, error) {
	value, ok := pd[PluginDefEmitJSONKey]
	if !ok || value == nil {
		return false, fmt.Errorf("%s was not provided in the plugin definition", PluginDefEmitJSONKey)
	}

	ret, ok := value.(*bool)
	if !ok {
		return false, fmt.Errorf("%s is not a bool", PluginDefEmitJSONKey)
	}

	return *ret, nil
}

// GetPluginDefNonzeroOnAnyErrorKey returns the EmitJSON flag
func (pd PluginDef) GetPluginDefNonzeroOnAnyErrorKey() (bool, error) {
	value, ok := pd[PluginDefNonzeroOnAnyErrorKey]
	if !ok {
		return false, nil
	}

	ret, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s is not a bool", PluginDefNonzeroOnAnyErrorKey)
	}

	return ret, nil
}

// PluginDef map
type PluginDef map[string]interface{}

// PluginOpts represents generic data that is given to a plugin
type PluginOpts interface{}

// PluginModule identifies the plax plugin by module name
type PluginModule string

// PluginMaker is the signature for a Plugin constructor
type PluginMaker func(def PluginDef) (Plugin, error)

// PluginRegistry maps a PluginModule to a constructor for that type of Plugin
type PluginRegistry map[PluginModule]PluginMaker

// Register registers a plugin
func (pr PluginRegistry) Register(module PluginModule, maker PluginMaker) {
	pr[module] = maker
}

// ThePluginRegistry is the global, well-know registry of supported plax Plugin types
var ThePluginRegistry = make(PluginRegistry)

// MakePlugin create the plugin
func MakePlugin(module PluginModule, def PluginDef) (Plugin, error) {
	maker, ok := ThePluginRegistry[module]
	if !ok {
		return nil, fmt.Errorf("plugin %s is not registered", module)
	}

	return maker(def)
}

// Plugin interface of invoking plax
type Plugin interface {
	// Invoke calls the plugin
	Invoke(ctx context.Context) error
}
