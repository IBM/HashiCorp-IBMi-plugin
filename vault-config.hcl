# Vault Configuration (Use your local absolute path)
storage "file" {
  path = "/Users/mohammedyaseenali/Documents/vault/data"
}

listener "tcp" {
  address     = "127.0.0.1:8200"
  tls_disable = 1
}

api_addr = "http://127.0.0.1:8200"
plugin_directory = "/Users/mohammedyaseenali/Documents/vault/plugins"
ui = true
log_level = "info"
disable_mlock = true
