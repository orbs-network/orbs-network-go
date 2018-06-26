package main

import (
  "fmt"
  "encoding/json"
)

func main () {
  var result = map[string]string{
    "hello": "world",
  }

  fmt.Println(result)

  jsonAsBytes, _ := json.Marshal(result)

  fmt.Println(string(jsonAsBytes))
}
