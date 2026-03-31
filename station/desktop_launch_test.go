package main

import "testing"

func TestShouldLaunchDesktopUI(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		parent string
		want   bool
	}{
		{name: "explorer double click", args: []string{"vessel.exe"}, parent: "explorer.exe", want: true},
		{name: "explorer path", args: []string{"C:\\tmp\\vessel.exe"}, parent: "C:\\Windows\\explorer.exe", want: true},
		{name: "open with launcher", args: []string{"vessel.exe"}, parent: "OpenWith.exe", want: true},
		{name: "unknown parent defaults to desktop", args: []string{"vessel.exe"}, parent: "customlauncher.exe", want: true},
		{name: "terminal no args", args: []string{"vessel.exe"}, parent: "powershell.exe", want: false},
		{name: "windows terminal host no args", args: []string{"vessel.exe"}, parent: "OpenConsole.exe", want: false},
		{name: "subprocess no args", args: []string{"vessel.exe"}, parent: "python.exe", want: false},
		{name: "explorer with cli args", args: []string{"vessel.exe", "list"}, parent: "explorer.exe", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldLaunchDesktopUI(tc.args, tc.parent); got != tc.want {
				t.Fatalf("shouldLaunchDesktopUI(%v, %q) = %v, want %v", tc.args, tc.parent, got, tc.want)
			}
		})
	}
}
