package main

import (
	"os"
	"os/signal"
	"syscall"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"

	"github.com/mannulus-immortalis/xmtask/internal/api"
	"github.com/mannulus-immortalis/xmtask/internal/api/auth"
	"github.com/mannulus-immortalis/xmtask/internal/db"
	"github.com/mannulus-immortalis/xmtask/internal/kafka"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// get config
	listen := os.Getenv("LISTEN_ADDRESS")
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		log.Fatal().Msg("DB_DSN env value is empty, see user manual for configuration description")
	}
	kafkaHost := os.Getenv("KAFKA_HOST")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	if kafkaHost == "" || kafkaTopic == "" {
		log.Fatal().Msg("Some env values for Kafka are missing, see user manual for configuration description")
	}
	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatal().Msg("JWT_KEY env value is empty, see user manual for configuration description")
	}

	// init deps
	dbConn, err := db.New(dbDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("DB connect failed")
	}
	defer dbConn.Close()

	jwtAuth, err := auth.New(jwtKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid HS256 key")
	}

	kafkaNotifier, err := kafka.New(&log, kafkaHost, kafkaTopic)
	if err != nil {
		log.Fatal().Err(err).Msg("kafka setup failed")
	}
	defer kafkaNotifier.Close()

	// setup API
	api := api.New(&log, dbConn, jwtAuth, kafkaNotifier)

	// run server in background
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- api.Run(listen)
	}()

	// listen to OS signals
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err = <-serverErrors:
		log.Err(err).Msg("received server error")
	case <-sig:
		log.Info().Msg("received shutdown signal")
		api.Close()
	}

}
