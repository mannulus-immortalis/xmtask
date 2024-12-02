package main

import (
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"

	"github.com/mannulus-immortalis/xmtask/internal/api/auth"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatal().Msg("JWT_KEY env value is empty, see user manual for configuration description")
	}

	if len(os.Args) < 2 {
		log.Fatal().Msg("missing command line parameter [role]")
	}
	roles := os.Args[1:]
	a, err := auth.New(jwtKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid HS256 key")
	}
	token, err := a.Generate(roles)
	if err != nil {
		log.Fatal().Err(err).Str("key", jwtKey).Msg("Token generation failed")
	}
	log.Info().Interface("Roles", roles).Str("JWT", token).Msg("JWT is genereated")
}
