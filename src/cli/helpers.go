package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/ALiwoto/codex-dedup/src/appStartup"
	"github.com/ALiwoto/codex-dedup/src/core/appValues"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitUsage
	}

	command := args[0]
	switch command {
	case "help", "-h", "--help":
		printUsage(stdout)
		return exitSuccess
	case "version", "--version":
		_, _ = fmt.Fprintf(stdout, "%s %s\n", appValues.ApplicationName, appValues.Version)
		return exitSuccess
	case string(appValues.ProxyRoleLocal), string(appValues.ProxyRoleRemote):
		return runProxyRole(appValues.ProxyRole(command), args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return exitUsage
	}
}

func runProxyRole(proxyRole appValues.ProxyRole, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet(string(proxyRole), flag.ContinueOnError)
	flags.SetOutput(stderr)
	configFile := flags.String("config", defaultConfigFile, "path to the INI configuration file")
	verbose := flags.Bool("verbose", false, "write informational logs to rotating files")
	flags.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "Usage: %s %s [options]\n", appValues.ApplicationName, proxyRole)
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitSuccess
		}
		return exitUsage
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected argument %q\n", flags.Arg(0))
		return exitUsage
	}
	// if !*check {
	// 	_, _ = fmt.Fprintf(stderr, "%s proxy runtime is not implemented yet; use --check to validate its configuration\n", proxyRole)
	// 	return exitFailure
	// }

	if err := appStartup.CheckConfig(&appStartup.ConfigCheckOptions{
		ProxyRole:  proxyRole,
		ConfigFile: *configFile,
		Verbose:    *verbose,
	}); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return exitFailure
	}

	_, _ = fmt.Fprintf(stdout, "%s configuration is valid\n", proxyRole)
	return exitSuccess
}

func printUsage(output io.Writer) {
	_, _ = fmt.Fprintf(output, `Usage: %s <command> [options]

Commands:
  local       configure the local proxy role
  remote      configure the remote proxy role
  version     print the application version
  help        print this help

The proxy runtimes are not implemented yet. Use local or remote with --check
to validate that role's configuration.
`, appValues.ApplicationName)
}
