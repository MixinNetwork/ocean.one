package config

import "encoding/json"

type MixinUser struct {
	ClientId        string `json:"id"`
	SessionId       string `json:"sid"`
	SessionKey      string `json:"key"`
	PinToken        string `json:"pin_token"`
	SessionAssetPIN string `json:"pin"`
	IsMaster        bool   `json:"is_master"`
}

var brokerMap = map[string]*MixinUser{}
var masterBroker *MixinUser

func init() {
	var brokers []*MixinUser
	err := json.Unmarshal([]byte(brokerJson), &brokers)
	if err != nil {
		panic("invalid broker json")
	}

	for _, broker := range brokers {
		if broker.IsMaster {
			masterBroker = broker
		}
		brokerMap[broker.ClientId] = broker
	}

	if masterBroker == nil {
		panic("no master broker")
	}
}

func Broker(brokerId string) *MixinUser {
	if broker, found := brokerMap[brokerId]; found {
		return broker
	}

	return nil
}

func MasterBroker() *MixinUser {
	return masterBroker
}
