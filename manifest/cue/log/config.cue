package log

#Level: "error" | "fatal" | "warn" |"info" | "debug"

#Config: {
    level: #Level
}

config: {
  level: *"error" | #Level
}
