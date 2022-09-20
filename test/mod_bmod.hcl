module "github.com/bmod" {
    priority = GLOBAL_BASE_PRIORITY_APP + 2
    disabled = false
    prefix   = "logging-"
    config {
        Filename                     = "${env2("HOME", ".")}/${GLOBAL_LOGDIR}/cmqd00.log"
        Append                       = GLOBAL_LOG_APPEND
        MaxBackups                   = anyname_MAX_BACKUPS
        RotateSchedule               = lower(anyname_ROTATE)
        DefaultLevel                 = GLOBAL_LOG_LEVEL
        DefaultPrefixWidth           = GLOBAL_LOG_PREFIX_WIDTH 
        DefaultEnableSourceLocation  = true
        Levels = [
            { Pattern="MCH_*", Level="DEBUG" },
            { Pattern="proc", Level="TRACE" },
            { Pattern="cemlib", Level="TRACE" },
        ]
    }
}
