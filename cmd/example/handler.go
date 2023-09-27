package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"sessions/pkg/sessions"
	"time"
)

var _userIdSesExtractor = sessions.SesExtractor(func(r *http.Request) sessions.SesId {
	return sessions.SesId(r.Header.Get("X-User-Id"))
})

const _expireTime = 1 * time.Minute

func Handler(redisClient *redis.Client) http.Handler {
	r := mux.NewRouter()
	r.Use(sessions.Middleware(_userIdSesExtractor, redisClient, _expireTime))
	r.HandleFunc("/my-profile", func(w http.ResponseWriter, r *http.Request) {
		profile, err := sessions.Get[UserProfile](r.Context(), "profile")
		if err != nil {
			if err == sessions.KeyNotFoundErr {
				profile = GetProfileFromSlowStorage()
				err = sessions.Set[UserProfile](r.Context(), "profile", profile)
				if err != nil {
					log.Printf("cannot save profile to session: %s", err)
				}

				_ = json.NewEncoder(w).Encode(profile)
				return
			}

			log.Printf("cannot get key from session: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(profile)
	})

	return r
}
