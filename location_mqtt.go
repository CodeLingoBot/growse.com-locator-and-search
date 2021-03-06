package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/lib/pq"
	"log"
	"time"
)

type MQTTMsg struct {
	Type                 string             `json:"_type" binding:"required"`
	TrackerId            string             `json:"tid"`
	Accuracy             int                `json:"acc"`
	Battery              int                `json:"batt"`
	Connection           string             `json:"conn"`
	Doze                 ConvertibleBoolean `json:"doze"`
	Latitude             float64            `json:"lat"`
	Longitude            float64            `json:"lon"`
	DeviceTimestampAsInt int64              `json:"tst" binding:"required"`
	DeviceTimestamp      time.Time
	Distance             float64
}

var topic = "owntracks/#"

func SubscribeMQTT(quit <-chan bool) error {
	log.Print("Connecting to MQTT")
	var mqttClientOptions = mqtt.NewClientOptions()
	if configuration.MQTTURL != "" {
		mqttClientOptions.AddBroker(configuration.MQTTURL)
	} else {
		mqttClientOptions.AddBroker("tcp://localhost:1883")
	}
	if configuration.MQTTUsername != "" && configuration.MQTTPassword != "" {
		mqttClientOptions.SetUsername(configuration.MQTTUsername)
		mqttClientOptions.SetPassword(configuration.MQTTPassword)
	}
	mqttClientOptions.SetClientID("growselocator")
	mqttClientOptions.SetAutoReconnect(true)
	mqttClientOptions.SetConnectionLostHandler(connectionLostHandler)
	mqttClientOptions.SetOnConnectHandler(onConnectHandler)
	mqttClient := mqtt.NewClient(mqttClientOptions)

	mqttClientToken := mqttClient.Connect()
	defer mqttClient.Disconnect(250)

	if mqttClientToken.Wait() && mqttClientToken.Error() != nil {
		log.Printf("Error connecting to mqtt: %v", mqttClientToken.Error())
		return mqttClientToken.Error()
	}
	log.Print("MQTT Connected")

	err := subscribeToMQTT(mqttClient, topic, handler)
	if err != nil {
		return err
	}

	select {
	case <-quit:
		log.Print("MQTT Unsubscribing")
		mqttUnsubscribeToken := mqttClient.Unsubscribe(topic)
		if mqttUnsubscribeToken.Wait() && mqttUnsubscribeToken.Error() != nil {
			log.Printf("Error unsubscribing from mqtt: %v", mqttUnsubscribeToken.Error())
		}
		log.Print("Closing MQTT")
		return nil
	}
}
func subscribeToMQTT(mqttClient mqtt.Client, topic string, handler mqtt.MessageHandler) error {
	log.Printf("MQTT Subscribing to %v", topic)
	mqttSubscribeToken := mqttClient.Subscribe(topic, 0, handler)
	if mqttSubscribeToken.Wait() && mqttSubscribeToken.Error() != nil {
		log.Printf("Error connecting to mqtt: %v", mqttSubscribeToken.Error())
		mqttClient.Disconnect(250)
		return mqttSubscribeToken.Error()
	}
	log.Printf("MQTT Subscribed to %v", topic)
	return nil
}

var onConnectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Print("MQTT Connected!")
	subscribeToMQTT(client, topic, handler)
}

var connectionLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("MQTT Connection lost: %v", err)
}

var handler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received mqtt message from %v", msg.Topic())
	var locator MQTTMsg
	err := json.Unmarshal([]byte(msg.Payload()), &locator)

	if err != nil {
		log.Printf("Error decoding MQTT message: %v", err)
		log.Print(msg.Payload())
		return
	}
	if locator.Type != "location" {
		log.Printf("Received message is of type %v. Skipping", locator.Type)
		return
	}

	newLocation := false

	locator.DeviceTimestamp = time.Unix(locator.DeviceTimestampAsInt, 0)
	dozebool := bool(locator.Doze)
	_, err = db.Exec(
		fmt.Sprintf(`
insert into locations
(timestamp,devicetimestamp,accuracy,doze,batterylevel,connectiontype,point) 
values ($1,$2,$3,$4,$5,$6, ST_GeographyFromText('SRID=4326;POINT(%[1]f %[2]f)'))`,
			locator.Longitude,
			locator.Latitude),
		time.Now(),
		&locator.DeviceTimestamp,
		&locator.Accuracy,
		&dozebool,
		&locator.Battery,
		&locator.Connection)

	switch i := err.(type) {
	case nil:
		newLocation = true
		break
	case *pq.Error:
		log.Printf("Pg error: %v", err)
		log.Printf("Locator struct: %v", locator)
	default:
		log.Printf("%T %v", err, err)
		InternalError(i)
		return
	}
	if newLocation {
		GeocodingWorkQueue <- true
	}
}
