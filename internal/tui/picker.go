package tui

type pickerOption struct {
	label    string
	enabled  bool
	subtitle string
}

var pickerOptions = []pickerOption{
	{"3x3x3", true, "two, three, and four-stage solvers"},
	{"4x4x4", false, "not supported yet"},
	{"2x2x2", false, "not present in this repository"},
}
