package kafka

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/mannulus-immortalis/xmtask/internal/models"
)

type kafka struct {
	log   *zerolog.Logger
	host  string
	topic string
	p     sarama.SyncProducer
}

func New(log *zerolog.Logger, host, topic string) (*kafka, error) {
	conf := sarama.NewConfig()
	conf.Producer.Return.Successes = true
	hosts := strings.Split(host, ",")
	p, err := sarama.NewSyncProducer(hosts, conf)
	if err != nil {
		return nil, err
	}

	return &kafka{
		log:   log,
		host:  host,
		topic: topic,
		p:     p,
	}, nil
}

func (k *kafka) Send(event models.EventNotifications) error {
	event.Timestamp = time.Now().Unix()
	data, err := json.Marshal(event)
	if err != nil {
		k.log.Err(err).Interface("Event", event).Msg("failed to marshal event")
		return err
	}

	producerMessage := &sarama.ProducerMessage{
		Topic: k.topic,
		Value: sarama.ByteEncoder(data),
	}
	_, _, err = k.p.SendMessage(producerMessage)
	if err != nil {
		k.log.Err(err).Msg("kafka send failed")
		return err
	}

	k.log.Info().Str("Topic", k.topic).Str("Message", string(data)).Msg("notification is sent to kafka")

	return nil
}

func (k *kafka) Close() {
	k.p.Close()
}
