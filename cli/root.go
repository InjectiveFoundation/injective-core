package cli

import (
	"fmt"

	"github.com/cosmos/gogoproto/grpc"
	"github.com/spf13/cobra"
)

// Arg is used to re-map and transform cli argument on index ArgIndex (when msg struct fields are not in order with cli command args)
type Arg struct {
	Index int

	// Transform is a functor to transform original string value from cli command into appropriate msg field value
	// Used when the values require special transformation besides simple parsing.
	// Returned string will be parsed as it was coming from cli inputs.
	// If Transform is omitted, we use original favue from arg.
	Transform func(origV string, ctx grpc.ClientConn) (tranformedV any, err error)
}

// Flag is used to re-map and transform cli flag named Flag
type Flag struct {
	Flag string

	// Transform is a functor to transform original string value from cli command into appropriate msg field value.
	// Used when the values require special transformation besides simple parsing.
	// Returned string will be parsed as it was coming from cli inputs.
	// If Transform is omitted, we use original favue from flag.
	Transform func(origV string, ctx grpc.ClientConn) (tranformedV any, err error)

	// UseDefaultIfOmitted changes the behavior of flag if it's not specified by the user.
	// If true, flag will return the default value.
	// If false, the struct's corresponding field will not be initialized with any value.
	UseDefaultIfOmitted bool
}

// special marker to skip parsing of the field from inputs and leave it zero initialized
var SkipField = Flag{Flag: ""}

// FlagsMapping maps message struct field names to corresponding flag names
type FlagsMapping map[string]Flag

// ArgsMapping maps message struct field names to corresponding arg inexes
type ArgsMapping map[string]Arg

// ModuleRootCommand generates module index root command
func ModuleRootCommand(moduleName string, isQuery bool) *cobra.Command {
	shortMsg := "Querying commands for the %s module"
	if !isQuery {
		shortMsg = "%s module transaction commands"
	}
	return &cobra.Command{
		Use:                        moduleName,
		Short:                      fmt.Sprintf(shortMsg, moduleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE: func(cmd *cobra.Command, args []string) error {
			usageTemplate := `Usage:{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}
  
{{if .HasAvailableSubCommands}}Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}
Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
			cmd.SetUsageTemplate(usageTemplate)
			return cmd.Help()
		},
	}
}
