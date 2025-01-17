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

//Package dphandler ..
package dphandler

import (
	"encoding/json"
	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	pluginConfig "github.com/ODIM-Project/ODIM/plugin-dell/config"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
)

var (
	// In Channel
	In chan<- interface{}
	// Out Channel
	Out <-chan interface{}
)

// RedfishEvents receives the subscribed events from the south bound system
// Then it will send the received data and ip to publish method
// RedfishEvents receives the subscribed events from the south bound system
// Then it will send the received data and ip to publish method
func RedfishEvents(w http.ResponseWriter, r *http.Request) {
	var req interface{}
	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	remoteAddr := r.RemoteAddr
	// if southbound entities are behind a proxy, then
	// originator address is expected to be in X-Forwarded-For header
	forwardedFor := r.Header.Get("X-Forwarded-For")
	forwardedFor = strings.Replace(strings.Replace(forwardedFor, "\n", "", -1), "\r", "", -1)
	if forwardedFor != "" {
		log.Info("Request contains X-Forwarded-For: " + forwardedFor + "; RemoteAddr: " + remoteAddr)
		addrList := strings.Split(forwardedFor, ",")
		// if multiple proxies are present, then the first address
		// in the X-Forwarded-For header is considered as originator address
		remoteAddr = addrList[0]
	}
	ip, _, err := net.SplitHostPort(remoteAddr)
	ip = strings.Replace(strings.Replace(ip, "\n", "", -1), "\r", "", -1)
	if err != nil {
		ip = remoteAddr
	}
	log.Debug("After splitting remote address, IP is: " + ip)

	request, _ := json.Marshal(req)

	reqData := string(request)
	//replacing the resposne with north bound translation URL
	for key, value := range pluginConfig.Data.URLTranslation.NorthBoundURL {
		reqData = strings.Replace(reqData, key, value, -1)
	}
	event := common.Events{
		IP:      ip,
		Request: []byte(reqData),
	}

	// Call writeEventToJobQueue to write events to worker pool
	writeEventToJobQueue(event)
	w.WriteHeader(http.StatusOK)
}

// writeEventToJobQueue will write events to worker pool
func writeEventToJobQueue(event common.Events) {
	var events []interface{}
	events = append(events, event)
	done := make(chan bool)
	go common.RunWriteWorkers(In, events, 5, done)
}
