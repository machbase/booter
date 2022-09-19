
module "github.com/machbase/cemlib/supervisor" {
    disabled = true
    priority = 10
    config = {
        config = "../../test/supervisor/config.ini"
    }
}