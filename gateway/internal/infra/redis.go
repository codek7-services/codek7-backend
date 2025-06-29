package infra

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)


var rdb *redis.Client


func RDBConnect() error {
  godotenv.Load()
  rdb = redis.NewClient(&redis.Options{
    Addr: os.Getenv("REDIS_HOST"),
    Password: os.Getenv("REDIS_PASSWORD"),
    DB: 0,
  })
  err := rdb.Ping(context.Background()).Err()
  if err != nil {
    return err
  }
  return nil
}

func GetRDB() *redis.Client {
  return rdb
}
