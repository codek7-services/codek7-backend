package main

import "github.com/lai0xn/codek-gateway/internal/infra"


func main(){
   
  err := infra.RDBConnect()
  if err != nil {
    panic(err)
  }
  err = infra.RMQConnect()
  if err != nil {
    panic(err)
  }
}
