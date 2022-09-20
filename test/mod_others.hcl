
module "github.com/machbase/cemlib/supervisor" {
    disabled = true
    priority = GLOBAL_BASE_PRIORITY_APP+10
    config {
        Config = "../../test/supervisor/config.ini"
    }
}