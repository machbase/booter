
## config syntax

### `define <prefix>`

개발환경/운영환경에 따라 수정이 필요한 사항들을
별도로 정의해두면 운영관리가 편리해진다.

아래와 같이 `define <prefix>` 형식으로 정의하며 prefix 부분은 임의의 문자열로 정의할 수 있고
값을 참조할 때에는 `prefix_name` 처럼 `_`로 연결되어 사용된다.

예를 들어 아래와 같이 정의 하였다면 `VARS_IP_ADDR` 은 "127.0.0.1"을 참조하게 된다.

```hcl
define VARS {
    IP_ADDR     = "127.0.0.1"
    DEBUG_MODE  = true
    MAX_BACKUPS = 10
    LOG_DIR     = "./logs"
}
```

- 정의의 이름은 관례상 대문자, 밑줄, 숫자로만 표현하도록 한다.
- 정의의 값은 문자열, 숫자, 불린(true, false) 상수와 함수 및 앞서 정의한 변수를 사용할 수 있다.
- define 블럭내에서 순서 상 뒤에 있는 다른 변수를 참조할 수 없다.
- 복수의 define 블럭이 존재할 경우 순서대로 정의된다.

### `module <moduleid>`

booter 프로세스 내에서 호출할 boot.Boot를 구현한 모듈들을 정의한다.

초기화하고 Start() 가 호출되는 순서는 기본적으로 파일에 기록된 순서대로이며
별도의 priority를 지정하면 해당 순서에 따라 Start()가 호출된다. Stop()은 역순으로 호출된다.

`module` 블럭에는 다음과 같은 값들을 설정할 수 있다.

- `name` 이름을 지정한다. 다른 module에서 `referece`로 depency injection을 수행할 때 이 이름으로 참조한다.

- `priority` 모듈을 시작하는 순서를 정수 값으로 지정한다. 작은 값일 수록 먼저 Start()된다.

- `diabled` 해당 모듈이 정의는 되어 있으나 인스턴스를 생성하거나 시작하지 않도록 disable한다.

- `inject <target> <field>` 해당 모듈을 대상(target) 모듈의 필드 (field)에 주입한다.
  주입되는 시점은 모든 모듈들의 인스턴스가 생성되고 Start()가 호출되기 이전이다.
  따라서 대상 모듈에서 Start()를 구현할 때 현재 Module의 인스턴스는 반드시 생성되어 있는 상태이지만 Start()가 된 상태인지는 순서에 대한 고려가 필요하다.

- `config` 해당 모듈의 config 객체를 정의한다.

module 정의 내에서는 위에서 `define`으로 정의한 변수와 미리 정의된 함수를 사용하여 구문을 작성할 수 있다.

```
module "my_project/module_a" {
    diabled = lower(VARS_IP_ADDR) == "127.0.0.1" ? true : false
    config {
        DebugMode      = VARS_DEBUG_MODE
        ListenAddress  = VARS_IP_ADDR 
        LogFilePath    = "${VARS_LOG_DIR}/my.log"
        HomePath       = env("HOME", "/home/my")
        Madatory       = envOrError("APP_VALUE")
    }
    inject "module_b" "ModB" {}
}

module "my_project/module_b" {
    name = "module_b"
}

```

#### functions
- `env(name, default)` 환경변수를 반환, 없으면 default를 반환한다. ex) `env("HOME", "/usr/home")`
- `envOrError(name)` 환경변수를 반환, 없으면 에러를 발생하고 booter가 종료된다. ex) `envOrError("APP_VALUE")`
- `flag(name, default)` 명령행 인자를 반환, 없으면 default를 반환한다. ex) `flag("--log-dir", "./tmp")`
- `flagOrError(name)` 명령행 인자를 반환, 없으면 에러를 발생하고 booter가 종료된다. ex)`flagOrError("--log-dir")`
- `pname()` booter 실행시 지정된 pname을 반환한다.
- `version()` 애플리케이션이 `booter.SetVersionString()`으로 설정한 값을 반환한다.
- `arg(i, default)` 명령행 인자들 중 플래그('-'로 시작하는)들을 제외한 i번째 인자를 반환, 없으면 default를 반환한다.
- `argOrError(i)` 명령행 인자들 중 플래그('-'로 시작하는)들을 제외한 i번째 인자를 반환, 없으면 에러를 발생한다.
- `arglen()` 명령행 인자 수를 반환
- `upper(str)`
- `lower(str)`
- `min(a, b)`
- `max(a, b)`
- `strlen(str)`
- `substr(str, offset, len)`

애플리케이션은 `booter.Startup()`을 호출하기 전에 `booter.SetFunction(name, function.Function)`으로 추가 함수를 정의할 수 있다.

> 일반적으로 응용프로그램에서는 설정값을 "기본값 -> 환경변수 -> 설정파일 -> 명령행 인자" 에서 참조하는데 다음과 같이 구현할 수 있다.
`MyPath = flag("--my-path", env("MY_PATH", "/home/me"))}`

### module 정의하기

직접 작성한 모듈을 booter의 config내에서 사용하기위해서는 booter가 시작되기 전에 booter의 레지스트리에 등록하는 절차가 필요하다. 일반적으로 init() 함수내에서 등록을 하도록 한다.

- `booter.Register(id, configFactory, instanceFactory)`를 호출하여 모듈을 등록한다.
- 여기서 세 개의 파라미터가 필요한데
- `id`는 해당 모듈의 식별자로 아무 문자열이나 상관없지만, 관례로 go module path를 사용한다.
- `configFactory`는 `func() T` 함수로 해당 모듈의 config 객체의 pointer `T`를 반환하도록 한다.
config 객체는 반환전에 default 값들을 채워서 config file에서 지정하지 않아도 디폴트값이 적용되도록 할 수 있다.
- `instanceFactory`는 `func(T) (boot.Boot, error)` 함수로 `configFactory`에서 반환한 객체에서
   설정파일의 `config` 블럭의 값들이 적용된 후 `instanceFactory`의 인자로 입력된다.
   이 값을 바탕으로 모듈의 인스턴스를 생성하여 반환하거나 오류를 반환하도록 한다.
   `instnaceFactory`의 반환 타입에서 알 수 있듯이 인스턴스는 `boot.Boot` 인터페이스를 구현해야 한다.
- `boot.Boot`는 `Start() error` `Stop()` 두 가지 함수를 가진 인터페이스이다.

### main() 정의하기

다음은 가장 단순형태의 booter application의 main()이다.

```go
func main() {
    booter.Startup()
    booter.WaitSignal()
    booter.Shutdown()
}
```

application이 `booter.Startup()`를 호출하여 booter를 시작하면
- booter는 config 파일들을 읽어들여 모듈들의 정의를 나열하고 
- 지정된 모듈들을 id를 기반으로 찾는다.
- 순서대로 해당 모듈의 configFactory를 호출하여 디폴트 config 객체를 받아온 후
- `config` 블럭에 지정된 필드들을 config 객체에 업데이트 한다.
- 수정된 config 객체를 instanceFactory에 전달하여 해당 모듈의 인스턴들을 생성한다.
- `reference` 블럭에 지정된 값을 기반으로 dependency injection을 수행한다.
- 각 모듈의 `Start()`를 순서대로 호출한다.

booter가 설정에 따라 application의 모듈들을 성공적으로 시작하였다면
`booter.WaitSignal()`을 호출하여 종료 시그널을 기다린다.
프로그램 제어 흐름은 `booter.WaitSignal()`에서 blocking된다.
이 상태에서 booter를 종료하려면 별도의 go routine에서 `booter.NotifySignal()`을 호출하면
`booter.WaitSignal()`이 반환되고 `booter.Shutdown()`을 통해 프로그램을 정상 종료하도록 한다.

#### csutomize command line arguments

`booter.Startup()`이 실행되면 다음과 같은 기본 명령행 인자를 바탕으로 실행된다.
이들 중에서 `--config-dir`, `--config` 두 가지 중 하나는 필수 항목이고 그 외의 플래그는 선택사항이다.
- `--config-dir <dir>` config directory path
- `-c, --config <file>` a single file config
- `--pname <name>` process name
- `--pid <path>` pid file path
- `--bootlog <path>` boot log path
- `--d, -daemon` run process in background, daemonize
- `--help` print this message

> 그 외의 응용프로그램에서 필요한 추가 플래그는 위의 functions에서 설명한 것처럼 설정파일 내에서 `flag()`로 지정하면 된다.

만약 이 파리미터를 변경해야할 필요가 있다면 `booter.Startup()`전에 `booter.SetFlag()`를 호출하여 변경할 수 있다.

```go
func SetFlag(flagType BootFlagType, longflag, shortflag, defaultValue string)

const (
	ConfigDirFlag
	ConfigFileFlag
	PnameFlag
	PidFlag
	BootlogFlag
	DaemonFlag
	HelpFlag
)
```

예를 들어 디폴트 플래그 `--config` 를 `--config-file`로 변경한다면 다음과 같이 한다. 
(shortflag를 사용하지 않으려면 "c" 대신 빈문자열 ""로 설정하면 된다.)
```go
booter.SetFlag(ConfigFileFlag, "config-file", "c", "./conf/default.hcl")
```

## Developer

### `.vscode/settings.json`

```json
{
    "files.exclude": {
        "vendor": true,
        "tmp": false,
        "bin": true,
    },
    "go.formatTool": "gofmt",
    "go.formatFlags": [
        "-s"
    ],
    "go.testFlags": [
        "-v",
        "-count", "1"
    ]
}
```