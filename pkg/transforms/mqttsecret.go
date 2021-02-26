//
// Copyright (c) 2020 Intel Corporation
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
//

package transforms

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/edgexfoundry/go-mod-core-contracts/clients"

	"github.com/epcom-hdxt/app-functions-sdk-go/appcontext"
	"github.com/epcom-hdxt/app-functions-sdk-go/pkg/secure"
	"github.com/epcom-hdxt/app-functions-sdk-go/pkg/util"
)

// MQTTSecretSender ...
type MQTTSecretSender struct {
	lock                 sync.Mutex
	client               MQTT.Client
	mqttConfig           MQTTSecretConfig
	persistOnError       bool
	opts                 *MQTT.ClientOptions
	secretsLastRetrieved time.Time
}

// MQTTSecretConfig ...
type MQTTSecretConfig struct {
	// BrokerAddress should be set to the complete broker address i.e. mqtts://mosquitto:8883/mybroker
	BrokerAddress string
	// ClientId to connect with the broker with.
	ClientId string
	// The name of the path in secret provider to retrieve your secrets
	SecretPath string
	// AutoReconnect indicated whether or not to retry connection if disconnected
	AutoReconnect bool
	// Topic that you wish to publish to
	Topic string
	// QoS for MQTT Connection
	QoS byte
	// Retain setting for MQTT Connection
	Retain bool
	// SkipCertVerify
	SkipCertVerify bool
	// AuthMode indicates what to use when connecting to the broker. Options are "none", "cacert" , "usernamepassword", "clientcert".
	// If a CA Cert exists in the SecretPath then it will be used for all modes except "none".
	AuthMode string
}

// NewMQTTSecretSender ...
func NewMQTTSecretSender(mqttConfig MQTTSecretConfig, persistOnError bool) *MQTTSecretSender {
	opts := MQTT.NewClientOptions()

	opts.AddBroker(mqttConfig.BrokerAddress)
	opts.SetClientID(mqttConfig.ClientId)
	opts.SetAutoReconnect(mqttConfig.AutoReconnect)
	//avoid casing issues
	mqttConfig.AuthMode = strings.ToLower(mqttConfig.AuthMode)
	sender := &MQTTSecretSender{
		client:         nil,
		mqttConfig:     mqttConfig,
		persistOnError: persistOnError,
		opts:           opts,
	}

	return sender
}

func (sender *MQTTSecretSender) initializeMQTTClient(edgexcontext *appcontext.Context) error {
	sender.lock.Lock()
	defer sender.lock.Unlock()

	// If the conditions changed while waiting for the lock, i.e. other thread completed the initialization,
	// then skip doing anything
	if sender.client != nil && !sender.secretsLastRetrieved.Before(edgexcontext.SecretProvider.SecretsLastUpdated()) {
		return nil
	}

	mqttFactory := secure.NewMqttFactory(
		edgexcontext.LoggingClient,
		edgexcontext.SecretProvider,
		sender.mqttConfig.AuthMode,
		sender.mqttConfig.SecretPath,
		sender.mqttConfig.SkipCertVerify,
	)

	client, err := mqttFactory.Create(sender.opts)
	if err != nil {
		return err
	}

	sender.client = client
	sender.secretsLastRetrieved = time.Now()

	return nil
}

func (sender *MQTTSecretSender) connectToBroker(edgexcontext *appcontext.Context, exportData []byte) error {
	sender.lock.Lock()
	defer sender.lock.Unlock()

	// If other thread made the connection while this one was waiting for the lock
	// then skip trying to connect
	if sender.client.IsConnected() {
		return nil
	}

	edgexcontext.LoggingClient.Info("Connecting to mqtt server for export")
	if token := sender.client.Connect(); token.Wait() && token.Error() != nil {
		sender.setRetryData(edgexcontext, exportData)
		subMessage := "dropping event"
		if sender.persistOnError {
			subMessage = "persisting Event for later retry"
		}
		return fmt.Errorf("Could not connect to mqtt server for export, %s. Error: %s", subMessage, token.Error().Error())
	}
	edgexcontext.LoggingClient.Info("Connected to mqtt server for export")
	return nil
}

// MQTTSend sends data from the previous function to the specified MQTT broker.
// If no previous function exists, then the event that triggered the pipeline will be used.
func (sender *MQTTSecretSender) MQTTSend(edgexcontext *appcontext.Context, params ...interface{}) (bool, interface{}) {
	if len(params) < 1 {
		// We didn't receive a result
		return false, errors.New("No Data Received")
	}

	exportData, err := util.CoerceType(params[0])
	if err != nil {
		return false, err
	}
	// if we havent initialized the client yet OR the cache has been invalidated (due to new/updated secrets) we need to (re)initialize the client
	if sender.client == nil || sender.secretsLastRetrieved.Before(edgexcontext.SecretProvider.SecretsLastUpdated()) {
		err := sender.initializeMQTTClient(edgexcontext)
		if err != nil {
			return false, err
		}
	}
	if !sender.client.IsConnected() {
		err := sender.connectToBroker(edgexcontext, exportData)
		if err != nil {
			return false, err
		}
	}

	token := sender.client.Publish(sender.mqttConfig.Topic, sender.mqttConfig.QoS, sender.mqttConfig.Retain, exportData)
	token.Wait()
	if token.Error() != nil {
		sender.setRetryData(edgexcontext, exportData)
		return false, token.Error()
	}

	edgexcontext.LoggingClient.Debug("Sent data to MQTT Broker")
	edgexcontext.LoggingClient.Trace("Data exported", "Transport", "MQTT", clients.CorrelationHeader, edgexcontext.CorrelationID)

	return true, nil
}

func (sender *MQTTSecretSender) setRetryData(ctx *appcontext.Context, exportData []byte) {
	if sender.persistOnError {
		ctx.RetryData = exportData
	}
}
