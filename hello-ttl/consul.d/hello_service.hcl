service {
  name = "hello"
  port = 8080
  checks = [
    {
      id = "hello-ttl"
      name = "5s TTL"
      notes = "Hello service notifies it is healthy every 5s seconds"
      ttl = "5s"
    }
  ]
}