/* (C) Copyright [2022] Hewlett Packard Enterprise Development LP
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may
 * not use this file except in compliance with the License. You may obtain
 * a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 */

// Package system ...

package system

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/config"
	aggregatorproto "github.com/ODIM-Project/ODIM/lib-utilities/proto/aggregator"
	"github.com/ODIM-Project/ODIM/svc-aggregation/agcommon"
	"github.com/ODIM-Project/ODIM/svc-aggregation/agmodel"
	"github.com/jarcoal/httpmock"
	"github.com/kataras/iris"
	"github.com/stretchr/testify/assert"
)

func Test_checkPluginStatus(t *testing.T) {
	phc := &agcommon.PluginHealthCheckInterface{
		DecryptPassword: common.DecryptWithPrivateKey,
	}
	password, _ := stubDevicePassword([]byte("password"))
	plugindata := agmodel.Plugin{
		IP:                "duphost",
		Port:              "9091",
		Username:          "admin",
		Password:          password,
		PreferredAuthType: "BasicAuth",
		ManagerUUID:       "mgr-addr",
	}

	type args struct {
		phc    *agcommon.PluginHealthCheckInterface
		plugin agmodel.Plugin
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				phc:    phc,
				plugin: plugindata,
			},
		},
	}
	ctx := mockContext()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkPluginStatus(ctx, tt.args.phc, tt.args.plugin)
		})
	}
}

func TestSendStartUpData(t *testing.T) {
	config.SetUpMockConfig(t)
	p := getMockExternalInterface()
	ctx := mockContext()
	pluginId := "GRF"
	defer removeMockPluginData(t, pluginId)
	mockPlugins(t)
	req := &aggregatorproto.SendStartUpDataRequest{
		PluginAddr: "1.1.1.1:1234",
		OriginURI:  "/mock",
	}
	resp := p.SendStartUpData(ctx, req)
	assert.Equal(t, nil, resp.Body, "Body should be nil")
}
func TestSendStartUpData_Success(t *testing.T) {
	config.SetUpMockConfig(t)
	p := getMockExternalInterface()
	ctx := mockContext()
	pluginId := "GRF"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://1.1.1.1:1234/ODIM/v1/Status",
		httpmock.NewStringResponder(200, `{Available:yes Uptime:361h11m56.037051983s TimeStamp:2023-04-14T09:50:21Z} EventMessageBus:{EmbType:Kafka EmbQueue:[{QueueName:REDFISH-EVENTS-TOPIC QueueDesc:Queue for redfish events}]}}`))
	mockPlugins(t)
	req := &aggregatorproto.SendStartUpDataRequest{
		PluginAddr: "1.1.1.1:1234",
		OriginURI:  "/mock",
	}
	agcommon.SetPluginStatusRecord("GRF", 2)
	resp := p.SendStartUpData(ctx, req)
	fmt.Println("resp", resp)
	removeMockPluginData(t, pluginId)
}

type mockHandlerFunc func(iris.Context)

func mockDeviceHandler(ctx iris.Context) {
	url := ctx.RequestPath(false)
	method := ctx.Method()

	resp, err := mockPluginStatus()
	if err != nil && resp == nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.WriteString("error: failed to change bios settings: " + err.Error())
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	ctx.Write(body)
	ctx.StatusCode(resp.StatusCode)
}
func mockPluginStatus() {

}
func mockHTTPSPlugins(handle) {
	irisHandler := iris.Handler(mockHandlerFunc)
	deviceApp := iris.New()
	deviceRoutes := deviceApp.Party("/redfish")
}
func mockPlugins(t *testing.T) {
	connPool, err := common.GetDBConnection(common.OnDisk)
	if err != nil {
		t.Errorf("error while trying to connecting to DB: %v", err.Error())
	}

	password := getEncryptedKey(t, []byte("Password"))
	pluginArr := []agmodel.Plugin{
		{
			IP:                "1.1.1.1",
			Port:              "1234",
			Password:          password,
			Username:          "admin",
			ID:                "GRF",
			PreferredAuthType: "BasicAuth",
			PluginType:        "GRF",
		},
	}
	for _, plugin := range pluginArr {
		pl := "Plugin"
		//Save data into Database
		if err := connPool.Create(pl, plugin.ID, &plugin); err != nil {
			t.Fatalf("error: %v", err)
		}
	}
}
func removeMockPluginData(t *testing.T, id string) {
	connPool, err := common.GetDBConnection(common.OnDisk)
	if err != nil {
		t.Errorf("error while trying to connecting to DB: %v", err.Error())
	}
	if err := connPool.Delete("Plugin", id); err != nil {
		t.Fatalf("error: %v", err)
	}

}
func TestPushPluginStartUpData(t *testing.T) {
	config.SetUpMockConfig(t)
	defer func() {
		err := common.TruncateDB(common.OnDisk)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		err = common.TruncateDB(common.InMemory)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
	}()
	startUpData := &agmodel.PluginStartUpData{
		RequestType:           "full",
		ResyncEvtSubscription: true,
	}
	plugin := agmodel.Plugin{
		ID: "10.0.0.0",
	}
	startUpData1 := &agmodel.PluginStartUpData{}
	ctx := mockContext()
	PushPluginStartUpData(ctx, agmodel.Plugin{}, startUpData)
	PushPluginStartUpData(ctx, agmodel.Plugin{}, startUpData1)

	err := PushPluginStartUpData(ctx, plugin, startUpData)
	assert.NotNil(t, err, "There should be no error")

}

func Test_sendPluginStartupRequest(t *testing.T) {
	config.SetUpMockConfig(t)
	defer func() {
		err := common.TruncateDB(common.OnDisk)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		err = common.TruncateDB(common.InMemory)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
	}()
	startUpData := agmodel.PluginStartUpData{
		RequestType:           "full",
		ResyncEvtSubscription: true,
	}
	ctx := mockContext()
	var startUpData1 interface{}
	_, err := sendPluginStartupRequest(ctx, agmodel.Plugin{}, startUpData1, "")
	assert.NotNil(t, err, "There should be error")
	_, err = sendPluginStartupRequest(ctx, agmodel.Plugin{}, startUpData1, "ILO_v2.0.0")
	assert.NotNil(t, err, "There should be error")
	_, err = sendPluginStartupRequest(ctx, agmodel.Plugin{}, startUpData, "ILO_v2.0.0")
	assert.NotNil(t, err, "There should be error")

}
func Test_sendFullPluginInventory(t *testing.T) {
	config.SetUpMockConfig(t)
	ctx := mockContext()
	defer func() {
		err := common.TruncateDB(common.OnDisk)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		err = common.TruncateDB(common.InMemory)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
	}()
	err := sendFullPluginInventory(ctx, "", agmodel.Plugin{})
	assert.Nil(t, err, "There should be no error")
	plugin := agmodel.Plugin{
		ID: "localhost",
	}

	mockPlugins(t)
	err = sendFullPluginInventory(ctx, "", plugin)
	assert.Nil(t, err, "There should be no error")

	err = sendFullPluginInventory(ctx, "10.0.0.0", plugin)
	assert.Nil(t, err, "There should be no error")

}

func Test_sharePluginInventory(t *testing.T) {
	config.SetUpMockConfig(t)
	defer func() {
		err := common.TruncateDB(common.OnDisk)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		err = common.TruncateDB(common.InMemory)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
	}()
	ctx := mockContext()
	sharePluginInventory(ctx, agmodel.Plugin{}, false, "")

	sharePluginInventory(ctx, agmodel.Plugin{}, false, "ILO_v2.0.0")
	sharePluginInventory(ctx, agmodel.Plugin{}, true, "ILO_v2.0.0")
}

func TestSendPluginStartUpData(t *testing.T) {
	config.SetUpMockConfig(t)
	defer func() {
		err := common.TruncateDB(common.OnDisk)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		err = common.TruncateDB(common.InMemory)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
	}()
	password := getEncryptedKey(t, []byte("Password"))

	plugin := agmodel.Plugin{
		IP:                "localhost",
		Port:              "1234",
		Password:          password,
		Username:          "admin",
		ID:                "GRF",
		PreferredAuthType: "BasicAuth",
		PluginType:        "GRF",
	}

	mockPlugins(t)
	ctx := mockContext()
	err := SendPluginStartUpData(ctx, "", agmodel.Plugin{})
	assert.Nil(t, err, "There should be no error")
	err = SendPluginStartUpData(ctx, "", plugin)
	assert.Nil(t, err, "There should be no error")

}
