package rtppf

import "flag"

type FlagSet struct {
	*flag.FlagSet
	Args []string
}

func (f *FlagSet) Bool(name string, def bool, usage string) *bool {
	return f.FlagSet.Bool(name, def, usage)
}

func (f *FlagSet) Int(name string, def int, usage string) *int {
	return f.FlagSet.Int(name, def, usage)
}

func (f *FlagSet) Float64(name string, def float64, usage string) *float64 {
	return f.FlagSet.Float64(name, def, usage)
}

func (f *FlagSet) String(name string, def string, usage string) *string {
	return f.FlagSet.String(name, def, usage)
}

func (f *FlagSet) BoolVar(pointer *bool, name string, def bool, usage string) {
	f.FlagSet.BoolVar(pointer, name, def, usage)
}

func (f *FlagSet) IntVar(pointer *int, name string, def int, usage string) {
	f.FlagSet.IntVar(pointer, name, def, usage)
}

func (f *FlagSet) Float64Var(pointer *float64, name string, def float64, usage string) {
	f.FlagSet.Float64Var(pointer, name, def, usage)
}

func (f *FlagSet) StringVar(pointer *string, name string, def string, usage string) {
	f.FlagSet.StringVar(pointer, name, def, usage)
}

func (f *FlagSet) StringList(name string, def string, usage string) *[]*string {
	return &[]*string{f.FlagSet.String(name, def, usage)}

}

func (f *FlagSet) ExtraUsage() string {
	return ""
}

func (f *FlagSet) Parse(usage func()) []string {
	f.FlagSet.Usage = func() {}
	f.FlagSet.Parse(f.Args)
	return f.FlagSet.Args()
}
