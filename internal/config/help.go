package config

import "fmt"

type HelpText struct {
	inParams  [2]string
	license   string
	helpShort string
}

func (help *HelpText) Init() {
	help.inParams = [2]string{"configfile", "configPath"}
}

func (help *HelpText) PrintLicense() {
	fmt.Println(help.GetHelpText())
}

func (help *HelpText) GetHelpText() string {
	return `test suite

Parameters for the executable --configfile <file name> --configpath <full path> --help

`
}
