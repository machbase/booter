
module "github.com/amod" {
    priority = 100
    config = {
        TcpConfig = {
            ListenAddress    = "${GLOBAL.IP_BIND}:1884"
            AdvertiseAddress = "${GLOBAL.IP_ADVERTISE}:1884"
            SoLinger         = 0
            KeepAlive        = 10
            NoDelay          = true
            Tls = {
                LoadSystemCAs    = false
                LoadPrivateCAs   = true
                CertFile         = GLOBAL.SERVER_CERT
                KeyFile          = GLOBAL.SERVER_KEY
                HandshakeTimeout = "5s" // equivalent 5000000000
            }
        }
    }
}
