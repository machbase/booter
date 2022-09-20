module "github.com/bmod" {
    priority = GLOBAL_BASE_PRIORITY_APP + 2
    disabled = false
    config {
        Filename                     = "${env2("HOME", ".")}/${GLOBAL_LOGDIR}/cmqd00.log"
        Append                       = GLOBAL_LOG_APPEND
        MaxBackups                   = anyname_MAX_BACKUPS
        RotateSchedule               = lower(anyname_ROTATE)
        DefaultLevel                 = flag("--logging-default-level", GLOBAL_LOG_LEVEL)
        DefaultPrefixWidth           = flag("--logging-default-prefix-width", GLOBAL_LOG_PREFIX_WIDTH)
        DefaultEnableSourceLocation  = flag("--logging-default-enable-source-location", true)
        Levels = [
            { Pattern="MCH_*", Level="DEBUG" },
            { Pattern="proc", Level="TRACE" },
            { Pattern="cemlib", Level="TRACE" },
        ]
    }
}
