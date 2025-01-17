//(C) Copyright [2020] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.

// Package umodel ...
package umodel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/errors"
	l "github.com/ODIM-Project/ODIM/lib-utilities/logs"
)

// Target is for sending the requst to south bound/plugin
type Target struct {
	ManagerAddress string `json:"ManagerAddress"`
	Password       []byte `json:"Password"`
	UserName       string `json:"UserName"`
	PostBody       []byte `json:"PostBody"`
	DeviceUUID     string `json:"DeviceUUID"`
	PluginID       string `json:"PluginID"`
}

// Plugin defines plugin configuration
type Plugin struct {
	IP                string
	Port              string
	Username          string
	Password          []byte
	ID                string
	PluginType        string
	PreferredAuthType string
}

// GetAllKeysFromTable fetches all keys in a given table
func GetAllKeysFromTable(table string, dbtype common.DbType) ([]string, error) {
	conn, err := common.GetDBConnection(dbtype)
	if err != nil {
		return nil, err
	}
	keysArray, err := conn.GetAllDetails(table)
	if err != nil {
		return nil, fmt.Errorf("error while trying to get all keys from table - %v: %v", table, err.Error())
	}
	return keysArray, nil
}

// GetResource fetches a resource from database using table and key
func GetResource(Table, key string, dbtype common.DbType) (string, *errors.Error) {
	conn, err := common.GetDBConnection(dbtype)
	if err != nil {
		return "", err
	}
	resourceData, err := conn.Read(Table, key)
	if err != nil {
		return "", errors.PackError(err.ErrNo(), "error while trying to get resource details: ", err.Error())
	}
	var resource string
	if errs := json.Unmarshal([]byte(resourceData), &resource); errs != nil {
		return "", errors.PackError(errors.UndefinedErrorType, errs)
	}
	return resource, nil
}

// GenericSave will save any resource data into the database
func GenericSave(ctx context.Context, body []byte, table string, key string) error {
	connPool, err := common.GetDBConnection(common.OnDisk)
	if err != nil {
		return fmt.Errorf("error while trying to connecting to DB: %v", err.Error())
	}
	if err = connPool.AddResourceData(table, key, string(body)); err != nil {
		if errors.DBKeyAlreadyExist == err.ErrNo() {
			return fmt.Errorf("error while trying to create new %v resource: %v", table, err.Error())
		}
		l.LogWithFields(ctx).Warn("skipped saving of duplicate data with key " + key)
	}
	return nil
}

// GetTarget fetches the System(Target Device Credentials) table details
func GetTarget(deviceUUID string) (*Target, *errors.Error) {
	var target Target
	conn, err := common.GetDBConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	data, err := conn.Read("System", deviceUUID)
	if err != nil {
		return nil, errors.PackError(err.ErrNo(), "error while trying to get compute details: ", err.Error())
	}
	if errs := json.Unmarshal([]byte(data), &target); errs != nil {
		return nil, errors.PackError(errors.UndefinedErrorType, errs)
	}
	return &target, nil
}

// GetPluginData will fetch plugin details
func GetPluginData(pluginID string) (Plugin, *errors.Error) {
	var plugin Plugin

	conn, err := common.GetDBConnection(common.OnDisk)
	if err != nil {
		return plugin, err
	}

	plugindata, err := conn.Read("Plugin", pluginID)
	if err != nil {
		return plugin, errors.PackError(err.ErrNo(), "error while trying to fetch plugin data: ", err.Error())
	}

	if err := json.Unmarshal([]byte(plugindata), &plugin); err != nil {
		return plugin, errors.PackError(errors.JSONUnmarshalFailed, err)
	}

	bytepw, errs := common.DecryptWithPrivateKey([]byte(plugin.Password))
	if errs != nil {
		return Plugin{}, errors.PackError(errors.DecryptionFailed, "error: "+pluginID+" plugin password decryption failed: "+errs.Error())
	}
	plugin.Password = bytepw

	return plugin, nil
}
