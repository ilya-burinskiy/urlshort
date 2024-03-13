package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ilya-burinskiy/urlshort/pkg/exitizer"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/staticcheck"
)

/*
Registered analyzers:

	    SA1000       Invalid regular expression
	    SA1001       Invalid template
	    SA1002       Invalid format in time.Parse
	    SA1003       Unsupported argument to functions in encoding/binary
	    SA1004       Suspiciously small untyped constant in time.Sleep
	    SA1005       Invalid first argument to exec.Command
	    SA1006       Printf with dynamic first argument and no further arguments
	    SA1007       Invalid URL in net/url.Parse
	    SA1008       Non-canonical key in http.Header map
	    SA1010       (*regexp.Regexp).FindAll called with n == 0, which will always return zero results
	    SA1011       Various methods in the 'strings' package expect valid UTF-8, but invalid input is provided
	    SA1012       A nil context.Context is being passed to a function, consider using context.TODO instead
	    SA1013       io.Seeker.Seek is being called with the whence constant as the first argument, but it should be the second
	    SA1014       Non-pointer value passed to Unmarshal or Decode
	    SA1015       Using time.Tick in a way that will leak. Consider using time.NewTicker, and only use time.Tick in tests, commands and endless functions
	    SA1016       Trapping a signal that cannot be trapped
	    SA1017       Channels used with os/signal.Notify should be buffered
	    SA1018       strings.Replace called with n == 0, which does nothing
	    SA1019       Using a deprecated function, variable, constant or field
	    SA1020       Using an invalid host:port pair with a net.Listen-related function
	    SA1021       Using bytes.Equal to compare two net.IP
	    SA1023       Modifying the buffer in an io.Writer implementation
	    SA1024       A string cutset contains duplicate characters
	    SA1025       It is not possible to use (*time.Timer).Reset's return value correctly
	    SA1026       Cannot marshal channels or functions
	    SA1027       Atomic access to 64-bit variable must be 64-bit aligned
	    SA1028       sort.Slice can only be used on slices
	    SA1029       Inappropriate key in call to context.WithValue
	    SA1030       Invalid argument in call to a strconv function
	    SA2000       sync.WaitGroup.Add called inside the goroutine, leading to a race condition
	    SA2001       Empty critical section, did you mean to defer the unlock?
	    SA2002       Called testing.T.FailNow or SkipNow in a goroutine, which isn't allowed
	    SA2003       Deferred Lock right after locking, likely meant to defer Unlock instead
	    SA3000       TestMain doesn't call os.Exit, hiding test failures
	    SA3001       Assigning to b.N in benchmarks distorts the results
	    SA4000       Binary operator has identical expressions on both sides
	    SA4001       &*x gets simplified to x, it does not copy x
	    SA4003       Comparing unsigned values against negative values is pointless
	    SA4004       The loop exits unconditionally after one iteration
	    SA4005       Field assignment that will never be observed. Did you mean to use a pointer receiver?
	    SA4006       A value assigned to a variable is never read before being overwritten. Forgotten error check or dead code?
	    SA4008       The variable in the loop condition never changes, are you incrementing the wrong variable?
	    SA4009       A function argument is overwritten before its first use
	    SA4010       The result of append will never be observed anywhere
	    SA4011       Break statement with no effect. Did you mean to break out of an outer loop?
	    SA4012       Comparing a value against NaN even though no value is equal to NaN
	    SA4013       Negating a boolean twice (!!b) is the same as writing b. This is either redundant, or a typo.
	    SA4014       An if/else if chain has repeated conditions and no side-effects; if the condition didn't match the first time, it won't match the second time, either
	    SA4015       Calling functions like math.Ceil on floats converted from integers doesn't do anything useful
	    SA4016       Certain bitwise operations, such as x ^ 0, do not do anything useful
	    SA4017       Discarding the return values of a function without side effects, making the call pointless
	    SA4018       Self-assignment of variables
	    SA4019       Multiple, identical build constraints in the same file
	    SA4020       Unreachable case clause in a type switch
	    SA4021       'x = append(y)' is equivalent to 'x = y'
	    SA4022       Comparing the address of a variable against nil
	    SA4023       Impossible comparison of interface value with untyped nil
	    SA4024       Checking for impossible return value from a builtin function
	    SA4025       Integer division of literals that results in zero
	    SA4026       Go constants cannot express negative zero
	    SA4027       (*net/url.URL).Query returns a copy, modifying it doesn't change the URL
	    SA4028       x % 1 is always zero
	    SA4029       Ineffective attempt at sorting slice
	    SA4030       Ineffective attempt at generating random number
	    SA4031       Checking never-nil value against nil
	    SA5000       Assignment to nil map
	    SA5001       Deferring Close before checking for a possible error
	    SA5002       The empty for loop ('for {}') spins and can block the scheduler
	    SA5003       Defers in infinite loops will never execute
	    SA5004       'for { select { ...' with an empty default branch spins
	    SA5005       The finalizer references the finalized object, preventing garbage collection
	    SA5007       Infinite recursive call
	    SA5008       Invalid struct tag
	    SA5009       Invalid Printf call
	    SA5010       Impossible type assertion
	    SA5011       Possible nil pointer dereference
	    SA5012       Passing odd-sized slice to function expecting even size
	    SA6000       Using regexp.Match or related in a loop, should use regexp.Compile
	    SA6001       Missing an optimization opportunity when indexing maps by byte slices
	    SA6002       Storing non-pointer values in sync.Pool allocates memory
	    SA6003       Converting a string to a slice of runes before ranging over it
	    SA6005       Inefficient string comparison with strings.ToLower or strings.ToUpper
	    SA9001       Defers in range loops may not run when you expect them to
	    SA9002       Using a non-octal os.FileMode that looks like it was meant to be in octal.
	    SA9003       Empty body in an if or else branch
	    SA9004       Only the first constant has an explicit type
	    SA9005       Trying to marshal a struct with no public fields nor custom marshaling
	    SA9006       Dubious bit shifting of a fixed size integer value
	    SA9007       Deleting a directory that shouldn't be deleted
	    SA9008       else branch of a type assertion is probably not reading the right value
	    appends      check for missing values after append
	    asmdecl      report mismatches between assembly files and Go declarations
	    assign       check for useless assignments
	    atomic       check for common mistakes using the sync/atomic package
	    atomicalign  check for non-64-bits-aligned arguments to sync/atomic functions
	    bodyclose    checks whether HTTP response body is closed successfully
	    bools        check for common mistakes involving boolean operators
	    buildssa     build SSA-form IR for later passes
	    buildtag     check //go:build and // +build directives
	    cgocall      detect some violations of the cgo pointer passing rules
	    composites   check for unkeyed composite literals
	    copylocks    check for locks erroneously passed by value
	    ctrlflow     build a control-flow graph
	    deepequalerrors check for calls of reflect.DeepEqual on error values
	    defers       report common mistakes in defer statements
	    directive    check Go toolchain directives such as //go:debug
	    errcheck     check for unchecked errors
	    errorsas     report passing non-pointer or non-error values to errors.As
	    exitizer     check for os.Exit calls in main function
	    fieldalignment find structs that would use less memory if their fields were sorted
	    findcall     find calls to a particular function
	    framepointer report assembly that clobbers the frame pointer before saving it
	    httpmux      report using Go 1.22 enhanced ServeMux patterns in older Go versions
	    httpresponse check for mistakes using HTTP responses
	    ifaceassert  detect impossible interface-to-interface type assertions
	    inspect      optimize AST traversal for later passes
	    loopclosure  check references to loop variables from within nested functions
	    lostcancel   check cancel func returned by context.WithCancel is called
	    nilfunc      check for useless comparisons between functions and nil
	    nilness      check for redundant or impossible nil comparisons
	    pkgfact      gather name/value pairs from constant declarations
	    printf       check consistency of Printf format strings and arguments
	    reflectvaluecompare check for comparing reflect.Value values with == or reflect.DeepEqual
	    shadow       check for possible unintended shadowing of variables
	    shift        check for shifts that equal or exceed the width of the integer
	    sigchanyzer  check for unbuffered channel of os.Signal
	    slog         check for invalid structured logging calls
	    sortslice    check the argument type of sort.Slice
	    stdmethods   check signature of methods of well-known interfaces
	    stringintconv check for string(int) conversions
	    structtag    check that struct field tags conform to reflect.StructTag.Get
	    testinggoroutine report calls to (*testing.T).Fatal from goroutines started by a test
	    tests        check for common mistaken usages of tests and examples
	    unmarshal    report passing non-pointer or non-interface values to unmarshal
	    unreachable  check for unreachable code
	    unsafeptr    check for invalid conversions of uintptr to unsafe.Pointer
	    unusedresult check for unused results of calls to some functions
	    unusedwrite  checks for unused writes
	    usesgenerics detect whether a package uses generics features
		errcheck     checks unchecked errors
		bodyclose    checks whenether res.Body is correctly closed
		exitizer     checks for os.Exic calls in main funcion
*/
func main() {
	analyzers := []*analysis.Analyzer{
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildssa.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		findcall.Analyzer,
		framepointer.Analyzer,
		httpmux.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		pkgfact.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
		errcheck.Analyzer,
		bodyclose.Analyzer,
		exitizer.Analyzer,
	}
	analyzers = append(analyzers, staticcheckAnalyzers()...)
	multichecker.Main(analyzers...)
}

func staticcheckAnalyzers() []*analysis.Analyzer {
	configFname := "staticcheck.json"
	appfile, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to configure staticcheck: %s\n", err.Error())
		return nil
	}

	data, err := os.ReadFile(filepath.Join(filepath.Dir(appfile), configFname))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read configs for staticcheck: %s\n", err.Error())
		return nil
	}

	type configData struct {
		Staticcheck []string
	}
	var config configData
	if err = json.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse staticcheck configs: %s\n", err.Error())
		return nil
	}
	checks := make(map[string]bool)
	for _, c := range config.Staticcheck {
		checks[c] = true
	}

	analyzers := make([]*analysis.Analyzer, 0, len(checks))
	for _, v := range staticcheck.Analyzers {
		if checks[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	return analyzers
}
