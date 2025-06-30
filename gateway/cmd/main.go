package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/lai0xn/codek-gateway/internal/infra"
	"github.com/lai0xn/codek-gateway/internal/server"
)


func main(){
  godotenv.Load()
  err := infra.RDBConnect()
  if err != nil {
    panic(err)
  }
  err = infra.RMQConnect()
  if err != nil {
    panic(err)
  }
  s := server.NewServer(os.Getenv("PORT"))
  s.Start()
}
