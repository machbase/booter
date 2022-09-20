
module "github.com/amod" {
    priority = GLOBAL_BASE_PRIORITY_APP + 1
    config {
        TcpConfig {
            ListenAddress    = "${GLOBAL_IP_BIND}:1884"
            AdvertiseAddress = "${GLOBAL_IP_ADVERTISE}:1884"
            SoLinger         = 0
            KeepAlive        = 10
            NoDelay          = true
            Tls {
                LoadSystemCAs    = false
                LoadPrivateCAs   = true
                CertFile         = GLOBAL_SERVER_CERT
                KeyFile          = GLOBAL_SERVER_KEY
                HandshakeTimeout = "5s" // equivalent 5000000000
            }
        }
    }
}
