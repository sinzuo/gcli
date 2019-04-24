package gcli

import (
	"strings"
)

// constants for error level 0 - 4
const (
	VerbQuiet uint = iota // don't report anything
	VerbError             // reporting on error
	VerbWarn
	VerbInfo
	VerbDebug
	VerbCrazy
)

// constants for hooks event, there are default allowed event names
const (
	EvtInit   = "init"
	EvtBefore = "before"
	EvtAfter  = "after"
	EvtError  = "error"
)

const (
	// OK success exit code
	OK = 0
	// ERR error exit code
	ERR = 2
	// HelpCommand name
	HelpCommand = "help"
)

/*************************************************************
 * CLI application
 *************************************************************/

// HookFunc definition.
// func arguments:
//  in app, like: func(app *App, data interface{})
//  in cmd, like: func(cmd *Command, data interface{})
// type HookFunc func(obj interface{}, data interface{})
type HookFunc func(obj ...interface{})

// Logo app logo, ASCII logo
type Logo struct {
	Text  string // ASCII logo string
	Style string // eg "info"
}

// App the cli app definition
type App struct {
	// internal use
	*CmdLine
	HelpVars
	SimpleHooks // allow hooks: "init", "before", "after", "error"

	// Name app name
	Name string
	// Version app version. like "1.0.1"
	Version string
	// Description app description
	Description string
	// Logo ASCII logo setting
	Logo Logo
	// Strict use strict mode. short opt must be begin '-', long opt must be begin '--'
	Strict bool
	// vars you can add some vars map for render help info
	// vars map[string]string
	// command names. key is name, value is name string length
	// eg. {"test": 4, "example": 7}
	names map[string]int
	// store some runtime errors
	errors []error
	// command aliases map. {alias: name}
	aliases map[string]string
	// all commands for the app
	commands map[string]*Command
	// all commands by module
	moduleCommands map[string]map[string]*Command
	// current command name
	commandName string
	// the max length for added command names. default set 12.
	nameMaxLength int
	// default command name
	defaultCommand string
	// clean os.args, not contains bin-name and command-name
	cleanArgs []string
}

// NewApp create new app instance.
// eg:
// 	New()
// 	// Or with a func.
// 	New(func(a *App) {
// 		// do something before init ....
// 		a.Hooks[gcli.EvtInit] = func () {}
// 	})
func NewApp(fn ...func(a *App)) *App {
	app := &App{
		Name: "My CLI App",
		Logo: Logo{Style: "info"},
		// set a default version
		Version:        "1.0.0",
		CmdLine:        CLI,
		commands:       make(map[string]*Command),
		moduleCommands: make(map[string]map[string]*Command),
		nameMaxLength:  12,
	}

	if len(fn) > 0 {
		fn[0](app)
	}

	// init
	app.Initialize()
	return app
}

// Config the application.
// Notice: must be called before adding a command
func (app *App) Config(fn func(a *App)) {
	if fn != nil {
		fn(app)
	}
}

// Initialize application
func (app *App) Initialize() {
	app.names = make(map[string]int)

	// init some help tpl vars
	app.AddVars(app.helpVars())

	// parse GlobalOpts
	// parseGlobalOpts()

	// add default error handler.
	app.SimpleHooks.Add(EvtError, defaultErrHandler)

	app.fireEvent(EvtInit, nil)
}

// SetLogo text and color style
func (app *App) SetLogo(logo string, style ...string) {
	app.Logo.Text = logo

	if len(style) > 0 {
		app.Logo.Style = style[0]
	}
}

// SetDebugMode level
func (app *App) SetDebugMode() {
	SetDebugMode()
}

// SetQuietMode level
func (app *App) SetQuietMode() {
	SetQuietMode()
}

// SetVerbose level
func (app *App) SetVerbose(verbose uint) {
	SetVerbose(verbose)
}

// DefaultCommand set default command name
func (app *App) DefaultCommand(name string) {
	app.defaultCommand = name
}

// NewCommand create a new command
func (app *App) NewCommand(name, useFor string, config func(c *Command)) *Command {
	return NewCommand(name, useFor, config)
}

// Add add one or multi command(s)
func (app *App) Add(c *Command, more ...*Command) {
	app.AddCommand(c)

	// if has more command
	if len(more) > 0 {
		for _, cmd := range more {
			app.AddCommand(cmd)
		}
	}
}

// AddCommand add a new command
func (app *App) AddCommand(c *Command) *Command {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		exitWithErr("The added command name can not be empty.")
	}

	if c.IsDisabled() {
		Logf(VerbDebug, "command %s has been disabled, skip add", c.Name)
		return c
	}

	i := strings.Index(c.Name, ":")
	if i == 0 {
		exitWithErr("The added command module can not be empty.")
	}

	if i > -1 {
		c.Module = c.Name[:i]
	} else {
		c.Module = ""
	}

	nameLen := len(c.Name)

	// add command to app
	app.names[c.Name] = nameLen
	app.commands[c.Name] = c

	// record command name max length
	if nameLen > app.nameMaxLength {
		app.nameMaxLength = nameLen
	}

	if _, ok := app.moduleCommands[c.Module]; !ok {
		app.moduleCommands[c.Module] = make(map[string]*Command)
	}
	app.moduleCommands[c.Module][c.Name] = c

	// add aliases for the command
	app.AddAliases(c.Name, c.Aliases)
	Logf(VerbDebug, "[App.AddCommand] add a new CLI command: %s", c.Name)

	// init command
	c.app = app
	c.initialize()
	return c
}

// On add hook handler for a hook event
func (app *App) On(name string, handler HookFunc) {
	Logf(VerbDebug, "[App.On] add application hook: %s", name)

	app.SimpleHooks.On(name, handler)
}

func (app *App) fireEvent(event string, data interface{}) {
	Logf(VerbDebug, "[App.Fire] trigger the application event: %s", event)

	app.SimpleHooks.Fire(event, app, data)
}

// AddError to the application
func (app *App) AddError(err error) {
	app.errors = append(app.errors, err)
}

// Names get all command names
func (app *App) Names() map[string]int {
	return app.names
}

// Commands get all commands
func (app *App) Commands() map[string]*Command {
	return app.commands
}

// CleanArgs get clean args
func (app *App) CleanArgs() []string {
	return app.cleanArgs
}
