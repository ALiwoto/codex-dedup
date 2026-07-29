# Agents Guide

## Go file layout
- Keep only one Go file in the repository root: `main.go`; rest of the files go under `src/` dir.
- Outside the root, follow the repository naming pattern for Go files:
  - `helpers.go` or `helpers_*.go`: only package-level helper functions
  - `methods.go` or `methods_*.go`: only methods
  - `types.go` or `types_*.go`: only type definitions
  - `constants.go` or `constants_*.go`: only `const` declarations
  - `vars.go` or `vars_*.go`: only `var` declarations
  - `handlers.go` or `handlers_*.go`: only functions or methods whose name contains `handle` (so `handler` is also fine).

- Keep new files aligned with the existing naming/content split instead of introducing mixed-purpose files like `startup.go`.

## Enforcement
- The repository enforces these rules with `./scripts/EnsureGoContent.ps1`.
- After changing Go files under `src`, run `./scripts/EnsureGoContent.ps1` and fix any violations before finishing.

## Folder structure
Prefer this folder structure:

- `ccl/`: in case you want to use ccl, put the .ccl files in here.
- `data/`: for caching data, if needed. (git-ignored, unless the data is needed to be tracked).
- `database/`: for db files, if needed. (git-ignored)
- `docs`: project docs.
- `logs`: output log files. (git-ignored)
- `src/`
  - `apiHandlers/`: all api handlers here
    - `binanceHandlers/`: binance-related handlers
      - `constants.go`
      - `handlers_ws.go`
      - `handlers.go`
      - `helpers_candles.go`
      - etc
    - `hlHandlers/`
      - same patterns here (only the files/things that we truly need)
    - etc
  - `core/`: put things that are truly shared among all the packages in here, not business-specific things
    - `appConfig/`: provides the current app's config to all packages
    - `appValues/`: values that need to be globally shared between packages
    - `cacheTypes/`: generated cache types, if we need many kind of cache types, create sub-dir in this
    - `exchangeLibs/`: these are used across the entire app
      - `binanceLib/`: binance related code
      - `crbLib/`
      - `hyperLib/`
      - `okxLib/`
    - `utils/`: all current app utils will go here. use sub-dir for different kind of utils
      - `aggressiveWs/`
      - `cacheUtils/`
      - `logging/`: very important! this is usually just a thin-wrapper over zap logger, but it's better for each repo to have this folder because we might want to customize logging mechanism per repo/project
    - `masterServer/`: this is where we load our "API server", bring up fiber, etc. if the app is supposed to bring up different api server, make two folders for it. e.g. `localServer` and `remoteServer`; and on the arg provided by the user decide to get which one up. Similarly, `apiHandlers` folder would also become two (again alongside, not nested): e.g. `localHandlers`, `remoteHandlers`
- `tests/`: tests folder (explained later).


## Function variables
- Never define a function as a package-level variable unless production code intentionally supports replacing or hooking that function at runtime.
- Do not add function variables solely so tests can replace production behavior. Keep test substitutions local to test code, and prioritize production-code clarity over test convenience.

## When to use methods
Prefer using methods when a function/operation heavily relies on a single struct and can be re-used:

```go
// bad!
func IsValid(container *MyContainer) bool {...}

// good:
func (c *MyContainer) IsValid() bool {...}

// bad!
func CloseSomething(something *SomeType, reason string, code int) error {...}

// good:
func (s *SomeType) CloseSomething(reason string, code int) error {...}

// bad!
func CloneSomeType(something *SomeType) *SomeType {...}

// good:
func (s *SomeType) Clone() *SomeType {...}
```

If doing something is truly and conceptually not re-usable and it's supposed to happen only and only at a single path ever in the code, then using methods for it is wrong:

```go
// wrong IF this validate method is only ever called in a single place in a single path in the code.
// if it's not truly *re-usable*, then adding it as a method would just bloat the struct. Prefer doing
// the validation in a helper method.
func (c *MyConfig) Validate() error {
    // lots of validation here
}

```

## Mutexes

### Unlock safety
Do not unlock mutexes without defer:

```go
// this is bad!
func DoSomething() {
    someMut.Lock()
    // some heavy logic here that can panic!
    someMut.Unlock() // if above code panics, the mutex will be locked forever!
}

// this is good:
func DoSomething() {
    someMut.Lock()
    defer someMut.Unlock() // it runs even in case of a panic

    // heavy logic here
}
```

### Mutex Exposing
Do not expose in-package and in-struct mutexes to public.

```go
var (
    // bad! other packages shouldn't know the existence of locs and how to manage them.
    // other packages should only know "helpers", "types", "methods" etc in this package, and those
    // are the ones that can manage these locks
    MyMutex = &sync.Mutex{}

    // good
    myMutex = &sync.Mutex{}
)

type StuffContainer {
    // bad! only this struct's method should know how/when to manage locks
    MyMut *sync.Mutex

    // good:
    mut *sync.Mutex

    // bad! do not embed mutexes, their methods will become public/exposed.
    *sync.Mutex
}

```

## init
Don't use `func init` for heavy logic. It's hard to debug and maintain and confusing to read and follow.
For tiny small things it's ok, such as:

```go
func init() {
    // bad! don't load API handlers here, we might need to catch errors, do heavy logic, etc
    loadAPIHandlers()

	// good: a tiny operation.
	_ = mime.AddExtensionType(".wasm", "application/wasm")
}
```

## Code Duplication
Don't do code **OR** logic duplication. Try to write re-usable code as much as possible.

However, "code de-duplication" does not only mean moving utils to shared/common util package, sometimes it's about the logic as well.
The code is **NOT** a black box. If you **doubt** a misuse of something basic can happen (e.g. uninitialized use of a struct), **just check the code**:

```go
// bad! we will keep repeating and duplicating this logic all the time....
func (s *Something) MethodA() {
    if s.mut == nil {
        s.Initialize()
    }
    // stuff
}
func (s *Something) MethodB() {
    if s.mut == nil || s.SomeField1 == nil {
        s.Initialize()
    }
    // stuff
}
func (s *Something) MethodC() {
    if s.mut == nil { // we forgot SomeField1 condition here...technical debt...
        s.Initialize()
    }
    // stuff
}
// and so on....this is all technical debt...

//-----------
// good: in helpers.go file, we have something like this:
func NewSomething() *Something { // can accept args too
    return *Something{
        mut: &sync.Mutex{},
        // or any other fields that need proper initialization
    }
}
func (s *Something) MethodB() {
    s.mut.Lock() // if someone misuses 
    defer s.mut.Unlock()
}

// things like this one are fine, as long as they are not constantly written for every single method:
func (s *Something) IsValid() bool {
    return s != nil && s.mut != nil && etc
}
```

Another form of code-duplication is the repeated call to sanitization of things, for example:

```go
// this is bad! we are constantly duplicating our supposedly **sanitization** here. what will happen if in future we need more sanitization??
func GetInternalObject(id string) *InternalObj {
    id = strings.TrimSpace(id)
    // other stuff
}
func DoSomething(id string) *InternalObj {
    id = strings.TrimSpace(id)
    // other stuff
}
func DoSomething2(id string) *InternalObj {
    id = strings.TrimSpace(id)
    // other stuff
}
```

To fix this specific issue, there is a very clever and preferable way: using Go's named type system:

```go
// This is nice and good and reliable.
type InternalId string

type (i InternalId) IsValid() bool {
    return strings.Contains(i, " ") // or other checks
}

// inside of GetInternalObject, DoSomething, DoSomething2, etc we will only accept InternalId
func GetInternalObject(id InternalId) *InternalObj {
    // we don't need to sanitize this here because we can safely assume it's a "valid" id, besides
    // this is just a Get method.
}

func ImportantSetObj(id InternalId, obj *InternalObj) error {
    // now this function is extremely important, we should **validate** it, not sanitize it
    if !id.IsValid() {
        return err // some error that represents the validation error
    }
}

// this is where we should **actually** sanitize it:
func NewInternalId(userInput string) InternalId { // you can also return error if we are also doing **validation** here
    id := strings.TrimSpace(userInput)
    return InternalId(id)
}

// or in case the sanitization is not that important or its logic is not that heavy, you can inline it (but it's less future-proof):

func SomeApiHandler(data SomeData) {
    internalId := InternalId(strings.TrimSpace(data.InternalId)) // this is better than constantly calling TrimSpace everywhere, because now we can actually and properly track where is an instance of `InternalId` is being generated! and managing/reading things is easy
    obj := GetInternalObject(internalId)
}
```

In general, prefer using named-types in Go for things that have **special concepts**, examples:

```go

// good: so if in future Go introduces things such as int128, we can just easily change a single line. also tracking them is easier.
type UserIdType int64
type SessionIdType string


// bad! these are better stay primitive types because their nature is truly primitive and they don't represent a special concept set:
type PageNumberType int64
type QueryLimitType int32
type FirstNameType string
```

## Simple data caching
For simple data caching which have known/unique keys, prefer using the normal disk-cache + memory-cache (on demand). Using an entire database such as SQLite or PostgreSQL just for simple caching is meaningless, unless you want to do specific operations on them such as range, grouping, search, specific indexing, etc...

You can also use `ccl`'s binary serialization codegen for storing caching.
If you want higher control, `bbolt` is also good.


## Tests
Do not add unit tests for every single tiny thing:

```go
// this is bad!
func TestHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"help"}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("failed")
	}
	if !strings.Contains(stdout.String(), "help stuff") {
		t.Fatalf("failed")
	}
}
```

example: Command-lines usually do not need unit tests, unless their output is truly important and will be used in some internal between softwares pipeline. otherwise they are just waste of test runtime.

- Do not write fragile tests, this is bad:

```go
func TestHelp(t *testing.T) {
    output := genSomething()
    if !strings.Contains(output, "some unreliable thing here") {
        t.Fatalf("failed")
    }
}
```

Because in future if the code inside of `genSomething` changes too much, lots of tests will fail. and we will need to do lots of fragile changes.

Prefer **conceptually correct tests**:

```go
func TestHelp(t *testing.T) {
    output := genSomething()
    if !strings.Contains(output, ourPackage.SomethingThatGetsUsedInOurPackage) {
        t.Fatalf("failed")
    }
}
```

- Do **NOT** hardcode custom URLs/domains, authorizations/tokens inside of the unit tests. Prefer mocking such tests or if they are explicitly required, make them read the values from env, and if they are not provided, just skip the test with a warning (not fail).

- If the project is supposed to be used in high concurrent situations, add aggressive race condition check for it.

- Always prefer writing tests inside of `./tests/` folder of the project instead of scattering them among the project file.
- Prefer the `./tests/` folder to have a proper file/sub-folder structure, preferably representing the real concepts of the functionalities you want to test. example:
  - `tests/`
    - `exchangeTests/`
      - `binance_ws_test.go`
      - `hyperliquid_ws_test.go`
      - `okx_ws_test.go`
- You don't have to follow the `helpers_*.go`, `types_*.go`, etc rule for the test files as long as they remain inside of the `./tests` folder.
- You don't have to write tests for **foundational** features inside of `./core/*` folder/sub-folders. E.g. no need to write tests for parsing configs or testing safe-maps, as they belong to the library itself (e.g. `ssg`).

## ssg
The ssg library is a shared library between all of my projects. It provides very common and useful utilities that can be used as foundation for many things:

### SafeMap
A map that will keep the pointer to values inside of it forever, unless explicitly removed:

```go
// initialize it in vars.go (or vars_*.go)
var myMap = ssg.NewSafeMap[string, MyType]()

//---

// get usage:
cacheData := myMap.Get(key) // nil if not found

cacheData := myMap.GetOrCreate(key, func() (*MyType, bool) {
    logging.Verbosef("created field1=%q key=%q", something, key)
    return &MyType{
        Field1: something,
    }, true
})

// or if you want ssg to handle a default pointer allocation for you:
cacheData := myMap.GetOrCreateDefault(key) // allocates &MyType{}
// can use `GetWithOptions` here too, explained later.

//---

// delete:
myMap.Delete(key)
myMap.DeleteIf(key, func(cacheItem *cacheUtils.SimpleCacheItem) bool {
    return cacheItem.IsExpired()
})
```

### SafeEMap:
same as SafeMap but its entries can expire if they are unused for a specific amount of time.
Example usages:

```go
// define this in a util package, e.g. src/core/utils/cacheUtils/helpers_map.go
func NewExpiringMap[TKey comparable, TValue any](
	expirationHour int,
	checkIntervalHour int,
	preExpFn func(key TKey, value *TValue) bool,
) *ssg.SafeEMap[TKey, TValue] {
	m := ssg.NewSafeEMap[TKey, TValue]()
	m.SetExpiration(time.Duration(expirationHour) * time.Hour)
	m.SetInterval(time.Duration(checkIntervalHour) * time.Hour)

	if preExpFn != nil {
		m.SetPreExpiringConditionFn(preExpFn)
	}
	m.EnableChecking()

	return m
}

// then inside of other packages, all you have to do is:
hlFundingFailures = cacheUtils.NewExpiringMap[string, HlFundingFailureData](1, 1, nil)

//---

// a more complex usage is specific locks map; used to properly manage mutex lifetimes:
func NewExpiringMutexMap[TKey comparable](
	expirationHour int,
	checkIntervalHour int,
) *ssg.SafeEMap[TKey, sync.Mutex] {
	return NewExpiringMap(
		expirationHour,
		checkIntervalHour,
		func(_ TKey, value *sync.Mutex) bool {
			if value == nil {
				return true
			}
			if !value.TryLock() {
				return false
			}

			value.Unlock()
			return true
		},
	)
}

func LockExpiringMutex[TKey comparable](
	mutexes *ssg.SafeEMap[TKey, sync.Mutex],
	key TKey,
) *sync.Mutex {
	return mutexes.GetWithOptions(
		key,
		&mapUtils.GetOptions[TKey, sync.Mutex]{
			CreateFn: func() (*sync.Mutex, bool) {
				return &sync.Mutex{}, true
			},
			DoFn: func(value *sync.Mutex) {
				value.Lock()
			},
		},
	)
}

// in a vars.go file:
var hlInfoReqMutexes  = cacheUtils.NewExpiringMutexMap[string](1, 1)

// real usage:
func HlInfoHandler(c *fiber.Ctx) error {
    // stuff

    mutKey := reqData.GetMutexKey()
	mut := cacheUtils.LockExpiringMutex(hlInfoReqMutexes, mutKey)
	defer mut.Unlock()

    // now we are properly putting a lock only on the correct mutex suitable for this request only
}
```


### Easy config parsing
`ssg.ParseConfig` uses proper go struct reflections to easily parse `.ini` config files:

```go
// file: src/core/appConfig/helpers.go

// you can also pass `config.ini:virtual` to support reading from environment vars
func LoadConfigFromFile(fileName string) error {
	if TheConfig != nil {
		return nil
	}
	var config = &PlatformConfig{}

	err := ssg.ParseConfig(config, fileName)
	if err != nil {
		return err
	}

	llmRuntimeEnabled.Store(config.LLMEnabled)
	parsedMinCandleTf, err := timeUtils.ParseDuration(config.MinCandleTf)
	if err == nil && parsedMinCandleTf >= time.Second {
		minCandleTf = parsedMinCandleTf
	}

	for i := range config.WhitelistedCandleAssets {
		config.WhitelistedCandleAssets[i] = strings.ToLower(
			strings.TrimSpace(config.WhitelistedCandleAssets[i]),
		)
	}

	TheConfig = config
	return nil
}

// file: src/core/appConfig/types.go

type PlatformConfig struct {
    SudoToken   string `section:"main" key:"sudo_token"`
	BindAddress string `section:"main" key:"bind_address" default:":4141"`

    CertFile    string `section:"main" key:"cert_file"`
	CertKeyFile string `section:"main" key:"cert_key_file"`

    // and so on....
}

```

## CCL (serialization codegen)
A simpler alternative to protobuf. If its CLI is installed on this machine (use `ccl help` to find out), prefer using it instead of JSON; because it generates strong-typed code, zero-reflection and its binary codec can save bandwidth. 

There is no protocol-versioning support in CCL (and we recommend you to not have it).
For internal-services that need to have a perfectly matching protocol, we recommend keeping `#[StrictBinaryParsing(true)]`; missing fields will result in error.

For backward-compatibility (e.g. cache storage), you can set `#[StrictBinaryParsing(false)]` and only add new fields to your model, this way the deserialization algorithm will act greedy and return back as much as it can read; but it must stay append-only, you can just mark the old fields as deprecated (and remove them gradually in future releases).

## Evidence-based guards and sanitization

Before adding a nil-check, fallback, sanitization, or other defensive branch, inspect every current construction site and caller. Add the branch only when the state can actually occur in the real world or when the value enters through an untrusted boundary where that state is valid input. Do not write branches for hypothetical future misuse.

```go
// bad: config is allocated immediately before this method call, and Validate
// has no other callers. The nil state cannot occur.
func (c *PlatformConfig) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	// ...
}

config := &PlatformConfig{}
if err := parseConfig(config); err != nil {
	return err
}
return config.Validate()

// good: rely on the constructor/call path that is already enforced.
func (c *PlatformConfig) Validate() error {
	// validate states that parsed user configuration can actually contain
}
```

The same rule applies to string sanitization. Confirm what the parser, constructor, or previous boundary already guarantees before adding `strings.TrimSpace`, case conversion, or similar transformations. `ssg.ParseConfig` already trims values read from INI files, so trimming every parsed config field again is duplication. Environment values are not trimmed by `ssg`; validate them exactly unless the application explicitly decides that environment input should be normalized.

```go
// bad: repeated speculative sanitization after the parser already did it.
config.LogDirectory = strings.TrimSpace(config.LogDirectory)
config.BindAddress = strings.TrimSpace(config.BindAddress)
config.ProviderURL = strings.TrimSpace(config.ProviderURL)

// good: validate the parsed values according to the operation that consumes them.
if config.LogDirectory == "" {
	return errors.New("log directory is required")
}
if _, _, err := net.SplitHostPort(config.BindAddress); err != nil {
	return fmt.Errorf("invalid bind address: %w", err)
}
```

If normalization is genuinely required, perform it once at the input boundary. For important concepts, prefer a named type and constructor so normalized values can be tracked instead of repeating sanitization throughout the codebase.

## Contextual type names

Type names must identify their domain clearly enough that a reader can understand most of their purpose without finding the declaration. Avoid vague names such as `Role`, `Options`, `Data`, or `Config` when the package can contain multiple kinds of those concepts.

```go
// bad: these names lose their meaning as soon as the package grows or the type
// is referenced without surrounding context.
type Role string
type Options struct {
	Verbose bool
}

// good: the names state which role and which operation they belong to.
type ProxyRole string
type ConfigCheckOptions struct {
	ProxyRole ProxyRole
	Verbose   bool
}
type LoggerOptions struct {
	Debug bool
}
```

Do not make names extremely long merely to encode every field. Include enough domain and operation context to prevent plausible confusion, such as mistaking a proxy role for a user authorization role.

## Option structs

Prefer passing option structs by pointer. Options commonly grow over time, should not be copied unnecessarily, and are naturally constructed for one operation. Name the type after that operation or subsystem rather than calling it only `Options`.

```go
// bad: vague type name and value copy.
type Options struct {
	ConfigFile string
}

func Check(options Options) error {
	// ...
}

// good: contextual name and pointer parameter.
type ConfigCheckOptions struct {
	ConfigFile string
}

func CheckConfig(options *ConfigCheckOptions) error {
	// ...
}
```

Do not automatically add `if options == nil` to a pointer-based options function. First inspect its callers. If all callers construct a non-nil options value and nil has no valid meaning, a nil guard is another unreachable defensive branch.


Sometimes, some callers are supposed to accept nil-options (basically when we want to say `"I do not want to override any options"` or when we want to do an operation "purely normal without any extra actions"). in which case, nil-options are allowed.
Example:

```go
type SomethingOptions struct {
    DoAnotherThing bool
}
func DoSomething(opts *SomethingOptions) error {
    if opts == nil {
        // opts is nil, so we will just choose the fast-path here
    }

    // we will do more advanced stuff here
}
```


