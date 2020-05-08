package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qjpcpu/common/debug"
)

type GotestArgs struct {
	Dir     string
	File    string
	IsDebug bool
}

func SelectSingleTest(dirname, file string, lastItem *Item) (name, fn string) {
	suites := LoadTestFiles(dirname, file)
	if suites.Size() == 0 {
		debug.Print("No tests found")
		return
	}
	suites = ReorderByHistory(suites, dirname, lastItem)
	if suites.Size() > 20 {
		suiteNames := suites.SuiteNames()
		_, name = debug.Select("Select test suite", suiteNames, func(s *debug.SelectWidget) {
			s.Size = 20
			s.IsVimMode = true
			s.HideSelected = true
			s.Searcher = func(input string, index int) bool {
				return strings.Contains(strings.ToLower(suiteNames[index]), strings.ToLower(input))
			}
		})
		if len(suites.SuiteFunctions(name)) > 0 {
			_, fn = debug.Select("Select test function", suites.SuiteFunctions(name), func(s *debug.SelectWidget) {
				s.Size = 20
				s.IsVimMode = true
				s.HideSelected = true
				s.StartInSearchMode = false
				fns := suites.SuiteFunctions(name)
				s.Searcher = func(input string, index int) bool {
					return strings.Contains(strings.ToLower(fns[index]), strings.ToLower(input))
				}
			})
		}
	} else {
		var list []string
		for _, n := range suites.SuiteNames() {
			fns := suites.SuiteFunctions(n)
			if len(fns) > 0 {
				for _, f := range fns {
					list = append(list, fmt.Sprintf("%s.%s", n, f))
				}
			} else {
				list = append(list, n)
			}
		}
		_, res := debug.Select("Select test function", list, func(s *debug.SelectWidget) {
			s.Size = 20
			s.IsVimMode = true
			s.HideSelected = true
			s.StartInSearchMode = false
			s.Searcher = func(input string, index int) bool {
				return strings.Contains(strings.ToLower(list[index]), strings.ToLower(input))
			}
		})
		if res == "" {
			return
		}
		debug.AllowPanic(func() {
			arr := strings.Split(res, ".")
			name = arr[0]
			fn = arr[1]
		})
	}
	return
}

func buildTestCommand(dir string, name, fn string, isDebug bool) string {
	var exe string
	if isDebug {
		exe = `dlv test -- `
	} else {
		exe = `go test `
	}
	format := "--test.run '^%s$' --testify.m '^%s$' --test.v"
	args := []interface{}{name, fn}
	if fn == "" {
		format = "--test.run '^%s$' --test.v"
		args = []interface{}{name}
	}
	format = exe + format

	wd, err := os.Getwd()
	debug.ShouldBeNil(err)
	wd, err = filepath.Abs(wd)
	debug.ShouldBeNil(err)
	dirAbs, err := filepath.Abs(dir)
	debug.ShouldBeNil(err)
	if wd != dirAbs {
		format = "cd '%s' && " + format
		if strings.HasPrefix(dir, wd) && len(wd) > 0 {
			dir = strings.TrimPrefix(dir, wd)
			if strings.HasPrefix(dir, "/") {
				dir = strings.TrimPrefix(dir, "/")
			}
		}
		args = append([]interface{}{dir}, args...)
	}
	debug.Print(format, args...)
	return fmt.Sprintf(format, args...)
}

func SelectAndRunTest(args GotestArgs) {
	item := History.Get(args.Dir)
	name, fn := SelectSingleTest(args.Dir, args.File, item)
	if len(name) == 0 {
		return
	}
	cmd := buildTestCommand(args.Dir, name, fn, args.IsDebug)
	History.Append(Item{Dir: args.Dir, Test: name, Module: fn})
	debug.Exec(cmd)
}

func getTestArgs(args []string) (targs GotestArgs) {
	if len(args) > 1 && args[1] == `debug` {
		targs.IsDebug = true
		args = args[1:]
	}
	const currentDir = "."
	if len(args) > 1 {
		targs.Dir = args[1]
		fi, err := os.Stat(targs.Dir)
		debug.ShouldBeNil(err)
		if !fi.IsDir() {
			targs.File = filepath.Base(targs.Dir)
			targs.Dir = filepath.Dir(targs.Dir)
		}
	} else {
		targs.Dir = currentDir
	}
	return
}
