module "github.com/bmod" {
    priority = 200
    disabled = true
    prefix   = "logging-"
    config = {
        filename                       = "${env2("HOME", ".")}/${GLOBAL.LOGDIR}/cmqd00.log"
        append                         = GLOBAL.LOG_APPEND
        max-backups                    = anyname.MAX_BACKUPS
        rotate-schedule                = lower(anyname.ROTATE)
        default-level                  = GLOBAL.LOG_LEVEL
        default-prefix-width           = GLOBAL.LOG_PREFIX_WIDTH 
        default-enable-source-location = false
        levels = [
            { pattern="MCH_*", level="DEBUG" },
            { pattern="proc", level="TRACE" },
            { pattern="cemlib", level="TRACE" },
        ]
    }
}
